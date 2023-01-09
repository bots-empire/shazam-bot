-- +goose Up
-- +goose StatementBegin
UPDATE shazam.users SET father_id = 0;
UPDATE shazam.users SET all_referrals = '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
