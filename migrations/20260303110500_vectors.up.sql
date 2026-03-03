-- +migrate Up
create extension if not exists vector;

create table if not exists vectors (
  id bigserial primary key,
  text text,
  image_url text,
  type text not null check (type in ('image', 'text')),
  vector vector not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_vectors_type on vectors (type);
