-- +migrate Up
alter table vectors
  add column if not exists text_vector vector,
  add column if not exists image_vector vector;

update vectors
set text_vector = vector
where type = 'text' and text_vector is null and vector is not null;

update vectors
set image_vector = vector
where type = 'image' and image_vector is null and vector is not null;
