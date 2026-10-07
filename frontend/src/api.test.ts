import { afterEach, describe, expect, it, vi } from "vitest";
import { buildQuery, searchListings } from "./api";
import type { SearchParams } from "./types";

const blank: SearchParams = {
  keyword: "", address: "", city: "", minPrice: "", maxPrice: "", minBedrooms: "",
  targetBudget: "", status: "", pageSize: "10", page: 1,
};

describe("buildQuery", () => {
  it("leaves out blank and whitespace-only fields", () => {
    const qs = buildQuery({ ...blank, city: "   ", keyword: "" });
    expect(qs.has("city")).toBe(false);
    expect(qs.has("keyword")).toBe(false);
    expect(qs.get("page")).toBe("1");
    expect(qs.get("pageSize")).toBe("10");
  });

  it("trims and passes through filled fields as typed", () => {
    const qs = buildQuery({ ...blank, city: " Vienna ", address: " main st ", minPrice: "400000", status: "active", page: 3 });
    expect(qs.get("city")).toBe("Vienna");
    expect(qs.get("address")).toBe("main st");
    expect(qs.get("minPrice")).toBe("400000");
    expect(qs.get("status")).toBe("active");
    expect(qs.get("page")).toBe("3");
  });

  it("does not validate on the client, so the server can explain bad input", () => {
    const qs = buildQuery({ ...blank, minPrice: "abc", pageSize: "0" });
    expect(qs.get("minPrice")).toBe("abc");
    expect(qs.get("pageSize")).toBe("0");
  });
});

describe("searchListings", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("rejects with AbortError when aborted while reading the body, instead of resolving null", async () => {
    const abort = new DOMException("The operation was aborted.", "AbortError");
    vi.stubGlobal("fetch", async () => ({ ok: true, status: 200, json: () => Promise.reject(abort) }));
    await expect(searchListings({ ...blank })).rejects.toMatchObject({ name: "AbortError" });
  });
});
