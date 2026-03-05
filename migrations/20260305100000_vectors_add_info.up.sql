-- +migrate Up
alter table vectors
  add column if not exists info text;
