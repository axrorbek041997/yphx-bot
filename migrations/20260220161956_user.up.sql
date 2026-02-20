-- +migrate Up
create table if not exists users (
  id serial primary key,
  tg_id bigint not null unique,
  username text unique,
  first_name text,
  last_name text,
  full_name text,
  phone text unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);