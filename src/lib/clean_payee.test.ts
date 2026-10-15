import { describe, it, expect } from "bun:test";
import { cleanPayee } from "./clean_payee";

describe("cleanPayee", () => {
  it("cleans complex Indian UPI and card statement strings", () => {
    expect(
      cleanPayee("UPI-628488132438-AXB20261011065534179345543681884180_UPI-628488132438-Smart Ba")
    ).toBe("Smart Ba");

    expect(
      cleanPayee("UPI-153753889520-UPI5ca7522e9efb5a2887d8e2ac5a2bf462_UPI-153753889520-ZEPTONOW")
    ).toBe("ZEPTONOW");

    expect(
      cleanPayee("UPI-628483076887-AXB20261011065534179345543681884170_UPI-628483076887-SWIGGY  ")
    ).toBe("SWIGGY");

    expect(
      cleanPayee("UPI-628476888036-RBA01a128edf6bbb180c3c81ac01df186f8_UPI-628476888036-Amazon I")
    ).toBe("Amazon I");

    expect(
      cleanPayee("UPI-628476639026-RBA01a1271bf68fe97d2e804990fc5898e2_UPI-628476639026-Amazon B")
    ).toBe("Amazon B");

    expect(
      cleanPayee("UPI-628476637684-APL01a1271ada1c694c5e121259e7805ae8_UPI-628476637684-Amazon I")
    ).toBe("Amazon I");

    expect(
      cleanPayee("UPI-174420692454-PPPL311920117010910261744056ac8da8d_UPI-174420692454-FLIPKART")
    ).toBe("FLIPKART");

    expect(cleanPayee("UPI-Smart Ba-14336279452-REF998877")).toBe("Smart Ba");

    expect(cleanPayee("UPI/SWIGGY/14333153570/swiggy@icici")).toBe("SWIGGY");

    expect(cleanPayee("BBPS Payment received 14238609506")).toBe("Payment received");
  });
});
