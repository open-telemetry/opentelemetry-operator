Deno.serve({
  port: 3000,
  onListen() {
    console.log("Hi");
  },
  handler(req) {
    return new Response("Hello World!");
  },
});
