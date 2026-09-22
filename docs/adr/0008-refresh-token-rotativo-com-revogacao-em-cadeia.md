# Refresh token rotativo, com revogação de toda a sessão do usuário no reuso

O login emite um JWT de acesso de 15 minutos e um refresh token opaco, persistido
**hasheado** em `refresh_tokens`. Cada uso do refresh token o invalida e emite o próximo
(rotação); reusar um token já rotacionado — sinal de que alguém além do usuário legítimo
tem uma cópia dele — revoga **todos** os refresh tokens ativos daquele usuário, não só a
cadeia à qual o token pertence.

Este é o desenho padrão de mercado para refresh token, mas "padrão de mercado" não é
motivo suficiente num sistema self-hosted de usuário único ou de um punhado de pessoas de
confiança ([ADR-0001](./0001-self-hosted-inquilino-unico.md)) — a escala que justifica a
complexidade em produtos multi-tenant não existe aqui. Vale registrar por que ainda assim
é a escolha certa, e não só copiada.

**Por que rotação, e não um refresh token único de vida longa.** Um refresh token que
nunca muda e vale por semanas ou meses é, na prática, uma segunda senha com validade maior
que a da própria senha, guardada em um cookie. Se o dispositivo for perdido, roubado ou
tiver malware, esse cookie sozinho autentica indefinidamente — sem rotação, não há nem
sinal de comprometimento nem forma de o sistema perceber. A rotação transforma "o token
vazou" de um evento silencioso em um evento detectável: o token antigo só volta a ser
apresentado se alguém tiver uma cópia paralela dele.

**Por que revogar a sessão inteira, e não só a cadeia comprometida.** A alternativa mais
comum é revogar apenas a família daquele refresh token, preservando as sessões dos outros
dispositivos do usuário. Rejeitamos essa versão mais cirúrgica aqui: com poucos
dispositivos por usuário (é um produto pessoal, não corporativo), o custo de um falso
positivo é baixo — logar de novo — e o benefício de forçar reautenticação em **todos** os
dispositivos é que o usuário necessariamente percebe o incidente, em vez de continuar
usando um outro dispositivo sem saber que uma cópia do seu refresh token circula em algum
lugar. Precisão cirúrgica que escondesse o sinal do próprio usuário seria otimizar a coisa
errada.

**Por que não pular rotação inteiramente**, aceitando um refresh token curto (poucos dias)
sem rotação: reduziria a implementação a quase nada, mas trocaria detecção por expiração —
um token roubado ainda funcionaria por toda a janela de validade, e é exatamente essa
janela que a insistência de cobrança deste produto torna sensível: uma sessão comprometida
que sobrevive dias tem acesso ao histórico de Compromissos e Ocorrências de alguém, o dado
mais pessoal que o sistema guarda. O custo de implementar rotação — uma tabela e uma
verificação de reuso — é pequeno comparado a esse risco, então não é complexidade que
"talvez nunca precise existir" no sentido do ADR-0001; é uma mitigação de custo fixo baixo
para um dado sensível.

## Consequências

- `refresh_tokens` guarda o hash do token, não o valor bruto — equivalente a senha para
  efeitos de armazenamento.
- Cada linha de `refresh_tokens` referencia a anterior (cadeia), para que o reuso de
  qualquer token da cadeia dispare a revogação; a revogação em si marca **todos** os
  tokens ativos do `user_id`, não apenas os da cadeia.
- Falso positivo esperado: múltiplas abas ou um retry de rede podem apresentar o mesmo
  refresh token duas vezes em corrida. Nesse caso o usuário é deslogado de todos os
  dispositivos e precisa entrar de novo — aceitável neste produto, não seria em um SaaS
  com muitos usuários simultâneos por conta.
- Se o produto crescer para múltiplos usuários por instância de forma mais séria, revisar
  se a revogação deveria ficar por dispositivo/cadeia em vez de por usuário — mas essa é
  uma migração de lógica, não de schema.
