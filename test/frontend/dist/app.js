// Test JavaScript file for compression testing
// This file should be large enough to trigger compression
console.log("Hello from SPA Server test app!");

const testData = {
    name: "spa-server",
    version: "1.0.0",
    features: [
        "zero-configuration",
        "spa-routing",
        "compression",
        "caching",
        "security-headers",
        "health-endpoints"
    ]
};

function init() {
    console.log("Initializing test app...");
    console.log(JSON.stringify(testData, null, 2));
}

init();
