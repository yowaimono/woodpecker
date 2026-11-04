export interface TeamActivityEvent {
  id: string;
  kind: 'commit' | 'merged_pr';
  repo_id: number;
  actor: string;
  actor_id: string;
  avatar: string;
  bot: boolean;
  title: string;
  url: string;
  occurred_at: string;
  created_at?: string;
}

export interface TeamActivityRepository {
  id: number;
  full_name: string;
  default_branch: string;
  status: 'pending' | 'complete' | 'partial' | 'skipped';
  issues: string[];
}

export interface TeamActivityReport {
  source: 'github';
  since: string;
  until: string;
  generated_at: string;
  partial: boolean;
  repositories: TeamActivityRepository[];
  events: TeamActivityEvent[];
}
