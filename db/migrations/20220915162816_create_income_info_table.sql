-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.income_info
(
    user_id bigint,
    source  text,
    UNIQUE(user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.income_info;
-- +goose StatementEnd
