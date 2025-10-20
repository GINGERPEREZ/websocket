const ws = new WebSocket(
  "ws://localhost:8080/ws/restaurant/1/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI5ZDA1NWJiOC1jMjc4LTRmYzUtYmJjNi1mNTAwNjU5MzMwYzUiLCJlbWFpbCI6InVzZXJAZXhhbXBsZS5jb20iLCJyb2xlcyI6WyJBRE1JTiJdLCJpYXQiOjE3NjA5MDg0MTMsImV4cCI6MTc2MDk5NDgxM30.tTu1JSiWZA2sK9hTox2w49BeChDOkT8ylOboED8WWCU"
);
ws.onopen = () => {
  console.log("✅ abierto");
  ws.send(
    JSON.stringify({
      action: "list_restaurants",
      payload: { limit: 1 },
    })
  );
};
ws.onmessage = (event) => console.log("📩", event.data);
ws.onclose = (event) => console.log("❌ cerrado", event.code, event.reason);
ws.onerror = console.error;
/*

AHora si haz lo mismo para las demas entidades del rest es decirl 
ws://localhost:8080/ws/table/:section/:token
ws://localhost:8080/ws/user/:section/:token
y asi con las demas entidades y tambien docuemntalos porfa. y que todo quede al 10000%
*/