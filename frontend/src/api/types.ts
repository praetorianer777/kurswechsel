export type Stance = "dafuer" | "dagegen" | "neutral" | "unklar";

export interface TopicSummary {
  slug: string;
  name: string;
  question: string;
  people: number;
  entries: number;
}

export interface PoliticianSummary {
  id: string;
  name: string;
  faction: string;
  role?: string;
  entries: number;
}

export interface FactionSpan {
  period: number;
  faction: string;
  from?: string;
  to?: string;
}

export interface TopicCount {
  slug: string;
  name: string;
  entries: number;
}

export interface PoliticianDetail {
  id: string;
  name: string;
  party?: string;
  is_mdb: boolean;
  factions: FactionSpan[];
  topics: TopicCount[];
}

export interface TimelineEntry {
  paragraph_id: number;
  speech_id: string;
  date: string;
  period: number;
  session: number;
  page?: number;
  faction?: string;
  role?: string;
  text: string;
  stance: Stance;
  quote?: string;
  rationale: string;
  confidence: number;
  model: string;
  source_url: string;
  change: boolean;
  previous_stance?: Stance;
}

export interface TimelineResponse {
  politician: PoliticianDetail;
  topic: { slug: string; name: string; question: string };
  changes: number;
  entries: TimelineEntry[];
}

export interface ReportRequest {
  paragraph_id: number;
  topic: string;
  message: string;
  contact: string;
}
