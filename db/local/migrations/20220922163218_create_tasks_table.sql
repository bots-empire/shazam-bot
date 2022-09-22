-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.tasks
(
    id              SERIAL,
    file_id         text,
    voice_length    int,
    UNIQUE(file_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.tasks;
-- +goose StatementEnd
