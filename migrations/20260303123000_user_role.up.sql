-- +migrate Up
alter table users
  add column if not exists role text not null default 'user';

do $$
begin
  if not exists (
    select 1
    from pg_constraint
    where conname = 'users_role_check'
  ) then
    alter table users
      add constraint users_role_check check (role in ('admin', 'user'));
  end if;
end $$;
