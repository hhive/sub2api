insert into settings (key, value, updated_at)
values
  ('onyx_enabled', 'true', now()),
  ('onyx_base_url', 'http://127.0.0.1:3000', now()),
  ('onyx_menu_label', 'Onyx', now()),
  ('onyx_exchange_secret', '1914d828a7188b5701d1bb979d43ac3d841f655564f77345d554f030f076d7e6', now()),
  ('onyx_launch_token_ttl_seconds', '60', now()),
  ('onyx_default_redirect_path', '/chat', now()),
  ('onyx_default_text_model', 'gpt-5.5', now()),
  ('onyx_default_image_model', 'gpt-image-2', now()),
  ('api_base_url', 'http://127.0.0.1:8080/v1', now())
on conflict (key) do update
set value = excluded.value,
    updated_at = now();
