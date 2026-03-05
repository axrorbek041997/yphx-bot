-- +migrate Down
alter table vectors
  drop column if exists info;
