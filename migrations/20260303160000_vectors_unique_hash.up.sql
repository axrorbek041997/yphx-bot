-- +migrate Up
alter table vectors
  add column if not exists text_norm text,
  add column if not exists text_hash text,
  add column if not exists image_hash text;

update vectors
set text_norm = nullif(lower(trim(regexp_replace(coalesce(text, ''), '\s+', ' ', 'g'))), '')
where type in ('text', 'image');

update vectors
set text_hash = md5(text_norm)
where text_norm is not null and text_hash is null;

update vectors
set image_hash = md5(image_url)
where type = 'image' and image_url is not null and image_hash is null;

delete from vectors v
using (
  select id
  from (
    select id, row_number() over (partition by text_hash order by id asc) as rn
    from vectors
    where type = 'text' and text_hash is not null
  ) t
  where t.rn > 1
) d
where v.id = d.id;

delete from vectors v
using (
  select id
  from (
    select id, row_number() over (partition by image_hash order by id asc) as rn
    from vectors
    where type = 'image' and image_hash is not null
  ) t
  where t.rn > 1
) d
where v.id = d.id;

create unique index if not exists ux_vectors_text_hash
  on vectors (text_hash)
  where type = 'text' and text_hash is not null;

create unique index if not exists ux_vectors_image_hash
  on vectors (image_hash)
  where type = 'image' and image_hash is not null;
