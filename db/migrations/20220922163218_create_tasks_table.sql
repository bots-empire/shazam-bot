-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.tasks
(
    file_id text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.tasks;
-- +goose StatementEnd
