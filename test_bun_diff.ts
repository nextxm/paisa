import { diffString } from "json-diff";

// Test string vs number in bun
const data = JSON.parse('{"amount": "20000"}');
const current = JSON.parse('{"amount": 20000}');
const diff = diffString(data, current);
console.log("diff:", JSON.stringify(diff));
console.log("empty?", diff === "");

// Test float
const data2 = JSON.parse('{"amount": "20000.5"}');
const current2 = JSON.parse('{"amount": 20000.5}');
const diff2 = diffString(data2, current2);
console.log("float diff:", JSON.stringify(diff2));
console.log("float empty?", diff2 === "");
