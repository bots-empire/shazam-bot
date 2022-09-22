-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS shazam;

CREATE TABLE IF NOT EXISTS shazam.users
(
    id              bigint,
    balance         int,
    completed       int,
    completed_today int,
    last_shazam     int,
    advert_channel  int,
    referral_count  int,
    take_bonus      boolean,
    lang            text,
    status          text,
	UNIQUE(id)
);

CREATE INDEX balanceindex ON shazam.users (balance);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA shazam CASCADE;
-- +goose StatementEnd
