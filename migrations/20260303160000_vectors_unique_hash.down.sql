-- +migrate Down
drop index if exists ux_vectors_text_hash;
drop index if exists ux_vectors_image_hash;

alter table vectors
  drop column if exists text_norm,
  drop column if exists text_hash,
  drop column if exists image_hash;
