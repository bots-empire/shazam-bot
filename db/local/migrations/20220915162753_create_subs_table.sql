-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.subs
(
    id bigint,
    UNIQUE(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.subs;
-- +goose StatementEnd
