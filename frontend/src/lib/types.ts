// Espelha os DTOs em backend/internal/adapters/http/*_handler.go.

export interface Area {
  id: string;
  name: string;
  color?: string;
}

export interface Label {
  id: string;
  name: string;
}

export type NagProfile = "leve" | "normal" | "implacavel";
export type CarryOverPolicy = "stack" | "replace";

export interface Commitment {
  id: string;
  area_id: string;
  title: string;
  description: string;
  recurrence_start: string;
  recurrence_rule?: string;
  nag_profile: NagProfile;
  carry_over_policy: CarryOverPolicy;
  archived_at?: string;
  label_ids: string[];
}

export type OccurrenceState =
  | "pending"
  | "due"
  | "nagging"
  | "done"
  | "snoozed"
  | "dismissed"
  | "abandoned";

export interface Occurrence {
  id: string;
  title: string;
  due_at: string;
  state: OccurrenceState;
  current_step: number;
}

export interface Settings {
  timezone: string;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  default_nag_profile: NagProfile;
  default_carry_over_policy: CarryOverPolicy;
}

export type ChannelKind = "email" | "whatsapp";

export interface Channel {
  id: string;
  kind: ChannelKind;
  address: string;
  verified: boolean;
  fallback_order: number;
}

export interface InstanceStatus {
  state: string;
}

export interface InstanceQRCode {
  qrcode: string;
}
