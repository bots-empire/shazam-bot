-- +goose Up
-- +goose StatementBegin
ALTER TABLE shazam.users ADD COLUMN father_id bigint;
ALTER TABLE shazam.users ADD COLUMN all_referrals text;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE shazam.users DROP COLUMN all_referrals;
ALTER TABLE shazam.users DROP COLUMN father_id;
-- +goose StatementEnd
