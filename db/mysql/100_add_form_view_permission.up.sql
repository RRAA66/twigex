UPDATE workspace_roles
SET permissions = JSON_ARRAY_APPEND(
    CASE
        WHEN permissions IS NULL OR TRIM(permissions) = '' OR NOT JSON_VALID(permissions)
        THEN '[]'
        ELSE permissions
    END,
    '$', 'views_form')
WHERE name IN ('user', 'admin')
  AND CASE
        WHEN JSON_VALID(permissions) THEN NOT JSON_CONTAINS(permissions, '"views_form"')
        ELSE 1
      END;
