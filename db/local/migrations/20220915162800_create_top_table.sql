-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.top
(
    top         int,
    user_id     bigint,
    time_on_top int,
    balance     int,
    UNIQUE(user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.top;
-- +goose StatementEnd
