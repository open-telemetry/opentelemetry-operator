import "./autoinstrumentation.js";
import module from "node:module";

if (typeof module.register === "function") {
    module.register("@opentelemetry/instrumentation/hooks.mjs");
} else {
    console.warn(
      `OpenTelemetry Operator auto-instrumentation could not instrument ESM code: Node.js ${process.version} does not support 'module.register()'`
    );
}
