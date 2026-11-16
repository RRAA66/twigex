UPDATE workspace_roles
SET permissions = TRIM(CONCAT(COALESCE(permissions, ''), ' views_form'))
WHERE name IN ('user', 'admin')
  AND CONCAT(' ', COALESCE(permissions, ''), ' ') NOT LIKE '% views_form %';
