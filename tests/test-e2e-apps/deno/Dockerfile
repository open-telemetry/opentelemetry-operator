FROM denoland/deno:alpine-2.1.5

COPY main.ts .

ENTRYPOINT ["deno", "-A", "--unstable-otel", "main.ts"]