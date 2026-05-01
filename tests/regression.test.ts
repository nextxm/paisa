import { spawn } from "bun";
import path from "path";
import net from "net";
import { describe, expect, test, beforeAll, afterAll } from "bun:test";
import waitPort from "wait-port";
import fs from "fs";
import axios from "axios";
import { diffString } from "json-diff";

const fixture = "tests/fixture";

function updateConfig(dir: string, from: string, to: string) {
  const filename = path.join(dir, "paisa.yaml");
  let config = fs.readFileSync(filename).toString();
  config = config.replace(from, to);
  fs.writeFileSync(filename, config);
}

const endpoints = [
  { route: "/api/dashboard", name: "dashboard" },
  { route: "/api/cash_flow", name: "cash_flow" },
  { route: "/api/income_statement", name: "income_statement" },
  { route: "/api/expense", name: "expense" },
  { route: "/api/recurring", name: "recurring" },
  { route: "/api/budget", name: "budget" },
  { route: "/api/assets/balance", name: "assets_balance" },
  { route: "/api/networth", name: "networth" },
  { route: "/api/investment", name: "investment" },
  { route: "/api/gain", name: "gain" },
  { route: "/api/allocation", name: "allocation" },
  { route: "/api/liabilities/balance", name: "liabilities_balance" },
  { route: "/api/liabilities/repayment", name: "liabilities_repayment" },
  { route: "/api/liabilities/interest", name: "liabilities_interest" },
  { route: "/api/income", name: "income" },
  { route: "/api/transaction", name: "transaction" },
  { route: "/api/editor/files", name: "files" },
  { route: "/api/ledger", name: "ledger" },
  { route: "/api/price", name: "price" },
  { route: "/api/diagnosis", name: "diagnosis" },
  { route: "/api/config", name: "config" }
];

describe("regression", () => {
  fs.readdirSync(fixture).forEach((dir) => {
    describe(dir, () => {
      const directory = path.join(fixture, dir);
      let fixturePort: number;
      let api: any;
      let proc: any;

      beforeAll(async () => {
        fixturePort = await new Promise<number>((resolve) => {
          const srv = net.createServer();
          srv.listen(0, () => {
            const port = (srv.address() as net.AddressInfo).port;
            srv.close(() => resolve(port));
          });
        });
        api = axios.create({ baseURL: `http://localhost:${fixturePort}` });

        const binary = process.platform === "win32" ? "./paisa.exe" : "./paisa";
        proc = spawn([
          binary,
          "--config",
          path.join(directory, "paisa.yaml"),
          "--port",
          fixturePort.toString(),
          "--now",
          "2022-02-07",
          "serve"
        ]);

        try {
          await waitPort({ port: fixturePort, output: "silent" });
        } catch (e) {
          // ignore
        }

        const {
          data: { job_id: jobId }
        } = await api.post("/api/sync", { journal: true });
        expect(jobId).toBeTruthy();

        // Poll until the background job reaches a terminal state (completed/failed).
        let jobStatus: string = "";
        for (let i = 0; i < 60; i++) {
          const { data: job } = await api.get(`/api/jobs/${jobId}`);
          jobStatus = job.status;
          if (jobStatus === "completed" || jobStatus === "failed") {
            break;
          }
          await Bun.sleep(500);
        }
        expect(jobStatus).toBe("completed");
      });

      afterAll(async () => {
        if (proc) {
          proc.kill();
          await proc.exited;
        }
      });

      endpoints.forEach((endpoint) => {
        test(endpoint.name, async () => {
          const { data } = await api.get(endpoint.route);
          const filename = path.join(directory, `${endpoint.name}.json`);
          const current = JSON.parse(fs.readFileSync(filename, "utf-8"));

          const diff = diffString(data, current, {
            excludeKeys: [
              "id",
              "transaction_id",
              "endLine",
              "transaction_end_line",
              "allow_legacy_auth",
              "disable_multi_currency_prices"
            ]
          });

          if (diff != "") {
            expect().fail(`Mismatch in ${endpoint.name}.json for fixture ${dir}:\n${diff}`);
          }
        });
      });
    });
  });
});
