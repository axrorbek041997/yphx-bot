-- +migrate Up
create table if not exists search_logs (
  id bigserial primary key,
  tg_user_id bigint not null,
  query_type text not null check (query_type in ('text', 'image', 'unknown')),
  query_text text,
  query_image_url text,
  result_text text,
  status text not null check (status in ('success', 'not_found', 'ignored')),
  reaction text check (reaction in ('like', 'dislike')),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists idx_search_logs_tg_user_id on search_logs (tg_user_id);
create index if not exists idx_search_logs_status on search_logs (status);
