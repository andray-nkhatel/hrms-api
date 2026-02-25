# Database Cleanup Scripts

## cleanup_employees.go

This script deletes all employees from the database except those with the `admin` role.

### What it does:
- Deletes all related data for non-admin employees:
  - Leaves
  - Leave accruals
  - Leave taken records
  - Leave carry over records
  - Identity information
  - Employment details
  - Employment history
  - Position assignments
  - Documents
  - Work lifecycle events
  - Onboarding/offboarding processes
  - Compliance records
  - Audit logs
  - Leave audits
- Deletes all non-admin employees
- Preserves all admin users

### How to run:

```bash
cd /home/andrea/Documents/Sources/hrms-api
go run scripts/cleanup_employees.go
```

The script will ask for confirmation. Type `DELETE` to proceed.

### ⚠️ WARNING:
This operation is **irreversible**. Make sure you have a database backup before running this script.

---

## enable-local-network-access.sh

Allows port 5173 (e.g. Vite dev server) through the firewall so other devices on your LAN can access it.

### 1. Allow the port (run once)

```bash
./scripts/enable-local-network-access.sh
```

### 2. Bind the dev server to all interfaces

Start your app so it listens on `0.0.0.0`, not only localhost:

- **Vite:** `npm run dev -- --host` or in `vite.config.ts`: `server: { host: true, port: 5173 }`
- **Other:** use the equivalent `--host 0.0.0.0` (or similar) for your dev server.

### 3. Open from other devices

Use your machine’s LAN IP and port 5173, e.g. `http://192.168.x.x:5173`. The script prints your IP when run.
