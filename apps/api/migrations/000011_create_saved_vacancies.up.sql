CREATE TABLE saved_vacancies (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    vacancy_id      BIGINT NOT NULL REFERENCES vacancies (id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, vacancy_id)
);

CREATE INDEX idx_saved_vacancies_user_id ON saved_vacancies (user_id);
