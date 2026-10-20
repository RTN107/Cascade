<p align="center">
  <img src="docs/screenshots/dashboard.png" alt="Cascade's HR Admin dashboard: tiles for 10 employees, 3 departments, 0 loss of pay not yet billed and the last payroll run's net pay in rupees, with a company-wide leave status bar and a headcount-by-department bar underneath" width="820" />
</p>

<p align="center">
  <a href="#1-try-it-in-two-minutes">Try it</a> •
  <a href="#2-what-you-need">What you need</a> •
  <a href="#3-setting-it-up">Setup</a> •
  <a href="#4-where-the-secrets-go">Secrets</a> •
  <a href="#5-making-it-yours">Making it yours</a> •
  <a href="#6-deploying-it-as-a-live-website">Deploying it</a> •
  <a href="#7-how-it-is-put-together">Structure</a> •
  <a href="#8-limits">Limits</a>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white" />
  <img alt="React" src="https://img.shields.io/badge/React-61DAFB?logo=react&logoColor=black" />
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-3178C6?logo=typescript&logoColor=white" />
  <img alt="PostgreSQL" src="https://img.shields.io/badge/PostgreSQL-4169E1?logo=postgresql&logoColor=white" />
  <img alt="Docker" src="https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white" />
</p>

# Cascade

An HR portal built around one real pattern: a reporting hierarchy that everything else,
visibility, leave approval, payroll deductions, reads live instead of storing separately. React
and TypeScript on the frontend, Go on the backend, PostgreSQL for data, the whole stack
containerized with Docker.

Who can see whom is a recursive query over a self-referencing manager column, computed on every
request, never cached into a separate table that can drift out of date. Leave balances pool
across every leave type an employee has into one combined cap for the year, so a request only
becomes unpaid once every type is actually exhausted, not the one type it happened to be filed
under. Payroll reads whatever pay components are configured per employee and whatever loss of
pay has not yet been billed, computed fresh on every run, with an all-or-nothing transaction if
anyone is missing a base salary.

**This repository ships empty.** No company, no people, no payroll numbers are built in. The
first account you create becomes the first HR Admin, and every department, employee, leave type
and pay component after that is entered through the app itself, by you. The screenshots on this
page show a fictional company, Brightloop Technologies, used only to demonstrate what the app
looks like with real-looking data in it; nothing about Brightloop is in the code.

---

## 1. Try it in two minutes

No company to invent yet, no cloud account.

```
git clone https://github.com/RTN107/Cascade.git
cd Cascade
cp .env.example .env
```

Open `.env` and set `JWT_SECRET` to any long random string, for example the output of
`openssl rand -hex 32`. Leave everything else as it is, then:

```
docker compose up --build
```

- API health check: http://localhost:8080/api/health
- Frontend: http://localhost:5173

The first time you open the frontend, the database is empty, so it asks you to create the
first HR Admin account instead of showing a login screen. That account can then add
departments, employees, leave types and pay components from inside the app. Everything below
is for understanding what you just started, and for putting it somewhere other than your own
machine.

---

## 2. What you need

| Requirement | What for |
|---|---|
| Docker and Docker Compose | Runs Postgres, the API and the frontend as three containers, nothing else to install |
| Go 1.26+ | Only if you want to run the API outside Docker |
| Node 22+ | Only if you want to run the frontend outside Docker |

There is no cloud account, no API key and no third-party service required to run this locally.
Section 6 covers the three free-tier services used to put it on the open internet.

---

## 3. Setting it up

### 3.1 Get the code

```
git clone https://github.com/RTN107/Cascade.git
cd Cascade
cp .env.example .env
```

### 3.2 Fill in `.env`

| Variable | What it controls | Can you leave it? |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | The local Postgres container's own login, only ever used between the `db` and `api` containers | Yes, the defaults work fine locally |
| `JWT_SECRET` | Signs the login session cookie | No, generate your own: `openssl rand -hex 32` |
| `API_PORT` | Which host port the API answers on | Yes, defaults to 8080 |

### 3.3 Run it

```
docker compose up --build
```

This starts three containers: `db` (Postgres), `api` (the Go server, which applies every SQL
migration in `backend/internal/db/migrations` on startup, so there is never a separate migrate
step to remember) and `web` (the Vite dev server). The first run builds the Go binary and
installs frontend dependencies, so it takes longer than later runs.

The frontend's host port, 5173, is fixed in `docker-compose.yml` and is not read from `.env`
the way `API_PORT` is. If you already have another copy of this project running, or anything
else on 5173, `docker compose up` fails with a port-in-use error; stop the other one first, or
change the `5173:5173` line in `docker-compose.yml`.

### 3.4 Create your company

Open http://localhost:5173. With no employees in the database yet, the app shows a setup
screen instead of a login screen: full name, email and password for the first account, which
is always created as HR Admin. From there, the People view adds departments and employees, the
Leave view adds leave types, and the Payroll view adds pay components and sets each employee's
salary structure. There is no seed script and no fixture data; this is the one and only way
data gets into a fresh database.

### 3.5 Running the pieces outside Docker

Only needed for editing the Go code with a debugger attached, or the frontend without the
container's file-watching overhead.

```
cd backend && go run ./cmd/api        # needs DATABASE_URL, JWT_SECRET set in your shell
cd frontend && npm install && npm run dev
```

The API still needs a reachable Postgres; `docker compose up db` on its own starts just that
container.

---

## 4. Where the secrets go

This repository contains no secrets. Everything lives in your own `.env`, which is git
ignored, or in the environment variables of whatever host you deploy to in section 6.

| Placeholder | File | Replace with |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | `.env` | any values, they only need to match between the `db` and `api` services |
| `JWT_SECRET` | `.env` | a long random string, for example `openssl rand -hex 32` |
| `API_PORT` | `.env` | defaults to 8080 if left blank |
| `WEB_ORIGIN` | `docker-compose.yml`, or an env var in production | the frontend's exact URL; the API only accepts requests from this one origin |
| `VITE_API_URL` | `docker-compose.yml`, or an env var in production | the API's exact URL, read by the frontend at build time |

No API keys, no third-party credentials and no real person's data are used anywhere in this
project. The only account with write access to anything is the one you create yourself in
step 3.4.

---

## 5. Making it yours

### 5.1 Roles

Four roles exist. The first three are the real hierarchy; the fourth is a deliberate exception.

| Role | Sees | Can write |
|---|---|---|
| `hr_admin` | Everyone, every department, every payslip | Everything: departments, employees, leave types, pay components, running payroll |
| `manager` | Their own department | Approves or rejects their reports' leave |
| `employee` | Their own chain of managers upward and their own reports downward | Only their own leave requests |
| `viewer` | Exactly what `hr_admin` sees | Nothing. Every write request from this role is refused in the authentication middleware itself, before it reaches any handler, regardless of what the frontend shows |

`viewer` exists for one reason: if you want to put a live, click-through demo of your own data
on the open internet, for a portfolio or a presentation, you can hand out one `viewer` login
and know that nobody who has it can change anything, change the password, or escalate out of
it. Create one the same way as any other employee, through the People view, choosing `viewer`
as the role.

<p align="center">
  <img src="docs/screenshots/org-tree.png" alt="Cascade's People view: a dashed card for the HR Director outside the reporting line, then a Customer Support department tree and an Engineering department tree, each with a manager and two reports" width="820" />
</p>

### 5.2 The reporting hierarchy

Nothing about departments, managers or reports is fixed in code. `employees.manager_id`
references another row in the same table, and every visibility and approval rule is a live
query over that one column, grouped by `department_id` for display. Reassign someone to a new
manager through the People view and every screen that depends on hierarchy, the org tree, who
can approve whose leave, who a manager's dashboard counts, reflects it on the next request.

### 5.3 Leave types and the pooled balance

Add as many leave types as you want, each with its own annual day balance, from the Leave
view. A request draws first from its own type, then from whichever other types still have
days left, pooling every type into one combined cap for the employee's year. Only days beyond
that whole pool become Loss of Pay. This is computed in `leave.Store.Decide` and recorded per
type in `leave_request_allocations`, so the balance shown per type is always the days actually
drawn from it, not an estimate.

<p align="center">
  <img src="docs/screenshots/team-calendar.png" alt="Cascade's team calendar for October 2026: a weekly grid with Ishaan shown as on Emergency leave on both the 21st and the 22nd" width="820" />
</p>

### 5.4 Pay components and the payroll formula

Basic Pay is the one required component, flagged `is_base`, and is the anchor every Loss of Pay
deduction is computed from. Add anything else your organization pays or deducts, HRA, a
Provident Fund contribution, a bonus, insurance, from the Pay components tab, then set each
employee's amount for it in Salary structure. A component can be removed later without changing
any payslip already generated; past payslips keep their own copy of the name and amount.

The name "Basic Pay" itself is seeded once by `backend/internal/db/migrations/0006_unify_salary.sql`,
not created through the UI, because exactly one component must be flagged `is_base` and the app
never exposes a way to set that flag. Renaming it to whatever your own payroll calls it, Base
Salary, Gaji Pokok, whatever the term is, means either editing that migration before the first
run, or running `UPDATE salary_components SET name = '...' WHERE is_base` yourself afterward.

<p align="center">
  <img src="docs/screenshots/pay-components.png" alt="Cascade's Pay components tab: Basic Pay required for every employee, and Employee's Contribution to Provident Fund as a removable deduction" width="820" />
  <img src="docs/screenshots/salary-structure.png" alt="Cascade's Salary structure tab for Ishaan Kapoor: Basic Pay 33000, Employee's Contribution to Provident Fund 3960, net pay 29040 before any Loss of Pay" width="820" />
</p>

One formula turns a day of Loss of Pay into a deduction amount, Basic Pay divided by a
days-per-month constant, and it is deliberately not an environment variable:

```go
// backend/internal/payroll/formula.go
const daysPerMonth = 22.0

func CalculateNetPay(basicPay, grossEarnings, totalDeductions, lopDays float64) (lopDeduction, netPay float64) {
	dailyRate := basicPay / daysPerMonth
	lopDeduction = math.Floor(dailyRate * lopDays)
	netPay = grossEarnings - totalDeductions - lopDeduction
	return lopDeduction, netPay
}
```

`daysPerMonth` is a working-days convention that genuinely differs between organizations, 22,
26 and 30 are all real choices elsewhere, so it is a constant you are expected to open and
change for your own convention rather than a setting with a wrong default. Everything else
about payroll, which components exist and what each employee is paid, is configured through
the app, never in code.

<p align="center">
  <img src="docs/screenshots/payslips-lop.png" alt="Cascade's HR Admin payslip table for October 2026 with Ishaan Kapoor's row expanded: Basic Pay added 33000, Provident Fund deducted 3960, Loss of pay 1 of 2 day(s) requested, 2026-10-21 to 2026-10-22, amount 1500" width="820" />
  <img src="docs/screenshots/payslip-pdf.png" alt="A generated payslip PDF for Ishaan Kapoor, October 2026: Basic Pay 33,000, Provident Fund deduction 3,960, Loss of pay 1,500 with the dates named, net pay 27,540" width="820" />
</p>

### 5.5 Branding

`frontend/src/components/Logo.tsx` and the colour variables in `frontend/src/App.css` are the
whole visual identity: one maroon mark, one serif display face (Zilla Slab) paired with a sans
body face (IBM Plex Sans), both loaded from Google Fonts in `frontend/index.html`. Change the
name in `index.html`'s `<title>`, the mark in `Logo.tsx`, and the colour variables; nothing in
the UI reads the word Cascade back out of an API response or a database row. The one place the
name used to leak past that was a `localStorage` key the collapsed-sidebar preference was saved
under, which is now just `rail-collapsed`, not tied to the product name.

### 5.6 Currency

Every amount in the app is Indian Rupees: `frontend/src/format.ts` prints the `₹` sign with
Indian digit grouping (lakhs and crores, `3,63,440` rather than `363,440`), and
`frontend/src/payslipPdf.ts` formats the generated PDF the same way so a figure never reads
differently on screen than it does in the download. Neither is a setting; both are two lines
in each file, written for a reader who expects INR. Change the symbol and the locale string in
both places for your own currency, together, so the two stay consistent with each other.

---

## 6. Deploying it as a live website

Running it locally is section 1. This section puts it on the open internet, on three free-tier
services: the database on [Neon](https://neon.tech), the API on [Render](https://render.com),
the frontend on [Vercel](https://vercel.com). None of the three need a card on file for a
project this size.

```
Vercel (frontend, Vite build of frontend/)
   |
   v  VITE_API_URL
Render (backend, Dockerfile in backend/)
   |
   v  DATABASE_URL
Neon (Postgres)
```

The backend reads every setting from environment variables and applies its own migrations on
startup, so a fresh deploy needs no manual database work beyond creating the database itself.

### 6.1 Database: Neon

1. Create a free Neon project and a database inside it.
2. Open the project, press **Connect**, and copy the pooled connection string. It looks like
   `postgresql://user:password@host/dbname?sslmode=require`.

### 6.2 Backend: Render

1. New Web Service, connect this repository, root directory `backend`, environment Docker (it
   picks up `backend/Dockerfile` automatically).
2. Choose the **Free** instance type.
3. Set these environment variables:

   | Variable | Value |
   |---|---|
   | `DATABASE_URL` | the Neon connection string from 6.1 |
   | `JWT_SECRET` | a long random string, for example `openssl rand -hex 32`, or press Render's own **Generate** button |
   | `WEB_ORIGIN` | the Vercel URL from 6.3, once you have it; leave a placeholder for now and come back |

4. Deploy. Render assigns a URL like `https://your-service.onrender.com` and injects its own
   `PORT` for the app to listen on, which `config.Load` falls back to automatically when
   `API_PORT` is not set. Check `https://<that-url>/api/health` returns `{"status":"ok"}`.

### 6.3 Frontend: Vercel

1. New Project, import this repository, root directory `frontend`. Vercel detects Vite
   automatically.
2. Set one environment variable: `VITE_API_URL` to the Render URL from 6.2, no trailing slash.
3. Deploy. Vercel gives you a URL like `https://your-project.vercel.app`.

### 6.4 Close the loop

Go back to Render and set `WEB_ORIGIN` to the exact Vercel URL from 6.3, then let it redeploy.
The API only accepts requests from this one origin, and this step is what makes that match the
frontend actually calling it.

**Cross-site cookies.** A frontend on `vercel.app` and an API on `onrender.com` are two
different sites as far as a browser is concerned, so the session cookie is set with
`SameSite=None; Secure` in `setSession` and `handleLogout`, which is what makes a browser send
it back on a cross-site request at all. If you ever put both behind the same domain instead,
this still works unchanged; it only stops working if you set `SameSite=Lax` back for a
same-site setup and then deploy cross-site anyway.

### 6.5 Share it

The link to give out is the Vercel URL. If this is a public demo, consider creating one
`viewer` account as in section 5.1 before sharing it, rather than handing out a real `hr_admin`
login.

Free instances on both Render and Neon spin down after a period of no traffic. The first
request after that can take 30 to 60 seconds while it wakes back up; after that it is instant
until it goes idle again. This is a free-tier tradeoff, not a bug.

---

## 7. How it is put together

```
Cascade/
├── backend/
│   ├── cmd/api/main.go              Entry point: load config, connect, migrate, serve
│   ├── internal/
│   │   ├── config/                  Every setting, read once from the environment
│   │   ├── db/
│   │   │   └── migrations/          Every schema change, applied once each, in order, on startup
│   │   ├── auth/                    Password hashing, session token issue and parse
│   │   ├── employees/               The hierarchy: departments, employees, who sees whom
│   │   ├── leave/                   Leave types, requests, the pooled balance, the team calendar
│   │   ├── payroll/                 Pay components, salary structure, the payroll run, the formula
│   │   ├── audit/                   One append-only log of who did what
│   │   └── httpapi/                 Every HTTP handler, the router, auth and role middleware
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── views/                   One file per screen: Dashboard, People, Leave, Payroll, Auth
│   │   ├── components/              Shell, the org tree, the team calendar, shared pieces
│   │   ├── api.ts                   The one place every HTTP request to the backend is made
│   │   ├── format.ts                Rupee and date formatting
│   │   └── payslipPdf.ts            Generates the payslip PDF shown in section 5.4
│   └── Dockerfile
├── docker-compose.yml               The three containers for section 1
├── .env.example                     Every variable section 4 explains
└── docs/screenshots/                The images used in this file
```

---

## 8. Limits

No rate limiting and no account lockout on repeated failed logins. The JWT-in-cookie session
has no refresh or revoke beyond its 24 hour expiry; there is no way to force-sign-out a
compromised session short of rotating `JWT_SECRET`, which signs everyone out at once.

Payroll's all-or-nothing transaction catches a missing Basic Pay, but does not otherwise
validate that an organization's pay components make business sense, negative amounts and
duplicate components are both possible through the UI.

The audit log records who did what, but nothing in the app currently reads it back; it is
write-only until a view is built for it.

Brightloop Technologies, shown in every screenshot on this page, is fictional. The company, its
people and its payroll numbers were invented to demonstrate the app and are not shipped with
this repository.
