import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import adminProxy from "./api/admin/[action].ts";
export default defineConfig(({ mode }) => {
  const privateEnv = loadEnv(mode, process.cwd(), "ORIX_");
  for (const [key, value] of Object.entries(privateEnv))
    process.env[key] ??= value;
  return {
    plugins: [
      react(),
      {
        name: "orix-local-admin-service",
        configureServer(server) {
          server.middlewares.use("/api/admin", (req, res, next) => {
            if (
              ![
                "/public-settings",
                "/android-upload-ticket",
                "/reports",
                "/report-thread",
                "/report-reply",
                "/resolve-report",
                "/support-tickets",
                "/support-thread",
                "/support-action",
                "/stats",
                "/lookup",
                "/logs",
                "/maintenance",
                "/jobs",
                "/failed",
                "/set-maintenance",
                "/set-version",
                "/user-role",
                "/restore-account",
                "/group",
                "/cancel-group",
                "/retry-delivery",
              ].includes(req.url?.split("?")[0] ?? "")
            ) {
              next();
              return;
            }
            void adminProxy(req, res);
          });
        },
      },
    ],
    build: { target: "es2022", manifest: true },
  };
});
