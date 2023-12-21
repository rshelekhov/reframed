CREATE TABLE IF NOT EXISTS users
(
    id         uuid PRIMARY KEY,
    email      character varying NOT NULL UNIQUE,
    password   character varying NOT NULL,
    role_id    int NOT NULL,
    first_name character varying NOT NULL,
    last_name  character varying NOT NULL,
    phone      character varying,
    created_at timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at timestamp WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS roles
(
    id    int PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    title character varying NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions
(
    id    int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    title character varying NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS user_roles
(
    user_id uuid,
    role_id int,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS role_permissions
(
    role_id int,
    permission_id int,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS appointments
(
    id                uuid PRIMARY KEY,
    doctor_id         uuid NOT NULL,
    client_id         uuid NOT NULL,
    title             character varying NOT NULL,
    content           text NOT NULL,
    status_id         int NOT NULL,
    scheduled_at      timestamp NOT NULL,
    first_appointment boolean DEFAULT false,
    created_by_id     uuid NOT NULL,
    created_at        timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at        timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at        timestamp WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_doctor_id ON appointments(doctor_id);
CREATE INDEX IF NOT EXISTS idx_client_id ON appointments(client_id);
CREATE INDEX IF NOT EXISTS idx_status_id ON appointments(status_id);

CREATE TABLE IF NOT EXISTS statuses
(
    id          int PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    status_name character varying NOT NULL
);

CREATE TABLE IF NOT EXISTS medical_reports
(
    id              uuid PRIMARY KEY,
    diagnosis       text NOT NULL,
    recommendations text NOT NULL,
    appointment_id  uuid NOT NULL,
    created_at      timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at      timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at      timestamp WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_appointment_id ON medical_reports(appointment_id);

CREATE TABLE IF NOT EXISTS attachments
(
    id                uuid PRIMARY KEY ,
    file_name         character varying NOT NULL,
    file_url          character varying NOT NULL,
    attachment_size   character varying NULL,
    medical_report_id uuid NOT NULL,
    attached_by_id    uuid NOT NULL,
    attached_at       timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at        timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at        timestamp WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_attached_by_id ON attachments(attached_by_id);

CREATE TABLE IF NOT EXISTS reminders
(
    id             uuid PRIMARY KEY,
    appointment_id uuid NOT NULL,
    user_id        uuid NOT NULL,
    content        character varying NOT NULL,
    read           boolean NOT NULL,
    created_at     timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at     timestamp WITH TIME ZONE NOT NULL DEFAULT now(),
    deleted_at     timestamp WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_appointment_id ON reminders(appointment_id);
CREATE INDEX IF NOT EXISTS idx_user_id ON reminders(user_id);

CREATE TABLE IF NOT EXISTS reminder_settings
(
    id       int PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    interval character varying NOT NULL
);

ALTER TABLE users ADD FOREIGN KEY (role_id) REFERENCES roles (id);
ALTER TABLE user_roles ADD FOREIGN KEY (user_id) REFERENCES users(id);
ALTER TABLE user_roles ADD FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE role_permissions ADD FOREIGN KEY (role_id) REFERENCES roles(id);
ALTER TABLE role_permissions ADD FOREIGN KEY (permission_id) REFERENCES permissions(id);
ALTER TABLE appointments ADD FOREIGN KEY (doctor_id) REFERENCES users (id);
ALTER TABLE appointments ADD FOREIGN KEY (client_id) REFERENCES users (id);
ALTER TABLE appointments ADD FOREIGN KEY (status_id) REFERENCES statuses (id);
ALTER TABLE appointments ADD FOREIGN KEY (created_by_id) REFERENCES users (id);
ALTER TABLE medical_reports ADD FOREIGN KEY (appointment_id) REFERENCES appointments (id);
ALTER TABLE attachments ADD FOREIGN KEY (medical_report_id) REFERENCES medical_reports (id);
ALTER TABLE attachments ADD FOREIGN KEY (attached_by_id) REFERENCES users (id);
ALTER TABLE reminders ADD FOREIGN KEY (appointment_id) REFERENCES appointments (id);
ALTER TABLE reminders ADD FOREIGN KEY (user_id) REFERENCES users (id);
