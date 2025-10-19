const ws = new WebSocket(
  "ws://localhost:8080/ws/restaurant/1/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI5ZDA1NWJiOC1jMjc4LTRmYzUtYmJjNi1mNTAwNjU5MzMwYzUiLCJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJyb2xlcyI6WyJBRE1JTiJdLCJpYXQiOjE3NjA5MDEzMDQsImV4cCI6MTc2MDk4NzcwNH0.qB-ZHkyhoxIFoQK0Hovj9ANvg1tOOo_7EEB44mQMi7Y"
);

ws.onopen = () => {
  console.log("✅ WebSocket abierto");
};

ws.onmessage = (event: MessageEvent) => {
  console.log("📩 Mensaje recibido:", event.data);
};

ws.onclose = () => {
  console.log("❌ WebSocket cerrado");
};

ws.onerror = (error) => {
  console.error("⚠️ Error en WebSocket:", error);
};
