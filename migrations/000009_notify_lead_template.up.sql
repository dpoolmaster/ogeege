INSERT INTO notify.templates (id, code, subject, body_md, channel, version, status)
VALUES (
    '01900000-0000-7000-8000-000000000003',
    'lead_notification',
    'Новая заявка с публичного сайта',
    E'Поступила новая заявка.\n\nИмя: {{.name}}\nКонтакт: {{.contact}}\nКомментарий: {{.comment}}\nИсточник: {{.source}}',
    'email',
    1,
    'active'
)
ON CONFLICT (code, version) DO NOTHING;
