import { describe, it, expect } from "bun:test";
import { DnsRecord } from "./dns-record";

describe("DnsRecord", () => {
  it("should create a DnsRecord provider", () => {
    expect(() => {
      new DnsRecord("TestDnsRecord", {
        zoneId: "test-zone-id",
        type: "A",
        name: "test.example.com",
        value: "192.168.1.1"
      });
    }).not.toThrow();
  });

  it("should have correct component type", () => {
    const record = new DnsRecord("TestDnsRecord", {
      zoneId: "test-zone-id",
      type: "A",
      name: "test.example.com",
      value: "192.168.1.1"
    });
    expect(record.constructor.name).toBe("DnsRecord");
  });

  it("should accept CNAME record configuration", () => {
    expect(() => {
      new DnsRecord("TestCNAME", {
        zoneId: "test-zone-id",
        type: "CNAME",
        name: "www.example.com",
        value: "example.com"
      });
    }).not.toThrow();
  });

  it("should accept CAA record with data configuration", () => {
    expect(() => {
      new DnsRecord("TestCAA", {
        zoneId: "test-zone-id",
        type: "CAA",
        name: "example.com",
        data: {
          flags: "0",
          tag: "issue",
          value: "letsencrypt.org"
        }
      });
    }).not.toThrow();
  });

  it("should accept proxied configuration", () => {
    expect(() => {
      new DnsRecord("TestProxied", {
        zoneId: "test-zone-id",
        type: "A",
        name: "proxied.example.com",
        value: "192.168.1.1",
        proxied: true
      });
    }).not.toThrow();
  });
});