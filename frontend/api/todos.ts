import { proxy } from "./_proxy";

export default proxy("/todos", ["GET", "POST"]);
