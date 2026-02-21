const { Pool } = require("pg");
const createApp = require("./app");

const pool = new Pool({
  connectionString:
    process.env.DATABASE_URL || "postgres://todo:todo@localhost:5432/todo",
});

const port = process.env.PORT || 3000;
const app = createApp(pool);

app.listen(port, () => {
  console.log(`Todo API listening on port ${port}`);
});
