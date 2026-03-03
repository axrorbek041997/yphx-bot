-- +migrate Down
alter table vectors
  drop column if exists text_vector,
  drop column if exists image_vector;
