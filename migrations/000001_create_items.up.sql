CREATE TABLE items (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT items_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT items_value_length CHECK (char_length(value) BETWEEN 1 AND 5000)
);
