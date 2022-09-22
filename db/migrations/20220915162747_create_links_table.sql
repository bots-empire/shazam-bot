-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS shazam.links
(
    hash        text,
    referral_id bigint,
    source      text
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS shazam.links;
-- +goose StatementEnd
