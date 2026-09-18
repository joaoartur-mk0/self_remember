# Self Remember - Uma solução para aqueles que tem mais tarefas do a mente consegue lembrar.

## **Ideia**

A ideia do Self Remember é unificar em um lugar em forma de CRM, todas as suas tarefas diárias, objetivos e metas, mas não para ficar em um painel de cards bonitos que você vai esquecer em 2 semanas e nunca mais vai abrir, e sim em uma solução que realmente te fará lembrar dos seus objetivos. A premissa do Self Remember não é apenas organizar, mas avisar, maior problema para se organizar hoje, não é ter o melhor KANBAN ou planilha, mas sim ter a melhor fuga das distrações e uma forma de burlar a falta de memória para o excesso de tarefas, e o Self Remember foca em solucionar isso, trazendo todas as integrações possíveis com os principais meios de comunicações ultilizados, Whatsapp, Email, Discord, Telegram ou Slack, permitindo que receba avisos dos seu compromissos no momento em que foi configurado, reduzido a chance de esquecer para 0, avisos que não pararam até que você va e marque como completo, do contrário, ainda serão recorrentes, garantido que o usuário se lembre do ocorrido.

## **Stack**

Backend: Go/Gin/Gorm/Postgrees/GoRiverQueue
Frontend: Vite/React/Tailwind

## **Ideias iniciais de integração*

1. EvolutionAPI: API mais acessível e opensource para fácil integração com o whatsapp.
2. Google Emails: API do Google Cloud para garantir que também receba os avisos pelo email.
3. STMP: API do SMTP para enviar emails.

## **CRM**

A ideia do CRM é que ele reflita diretamente nos avisos, podendo separar oque é relacionando ao trabalho dos objetivos pessoais, criar classificações, diferenciar tarefas diárias de tarefas únicas, tudo unido em um painel simples e de fácil uso, para que a memória não seja um problema e o usuário consiga se organizar com eficiência, além de poder alimentar o CRM com todo tipo de dados, para ter uma visão completa de tudo o que precisa ser feito.

## **Deploy**

Para deploy, será usado Docker Compose, um docker compose único que contem os serviços do banco de dados, front e backend, além de outros serviços necessários para o deploy, tendo a versão local e prod, local para testes e prod para deploy. Local será simples, sem configurações complexas, usando as portas para acesso local e sem configuração de label traefik. O prod terá configuração robusta com logging otimizado, usando labels traefik para roteamento e monitoramento e sem abrir portas, tudo via traefik.

## **Frontend**

Simples, objetivo de ser uma interface de usuário intuitiva e fácil de usar, para que o usuário possa se organizar e gerenciar suas tarefas.

## **Backend**

Simples, objetivo de ser uma API RESTful segura e escalável, que fornece os dados necessários para o frontend, além de realizar as operações CRUD (Create, Read, Update, Delete) nas tarefas. 

## **Documentação**

Documentação clara e detalhada, usando Swagger para gerar a documentação da API. No repositório manter documentação de mudanças com o CHANGELOG.md.
