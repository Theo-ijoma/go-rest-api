import { proxy } from "./_proxy.js";

export default proxy("/todos", ["GET", "POST"]);
