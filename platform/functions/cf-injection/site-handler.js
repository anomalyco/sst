import cf from "cloudfront";
async function handler(event) {
  __SST_CF_INJECT__();

  async function routeSite(kvNamespace, metadata) {
    const baselessUri = metadata.base
      ? event.request.uri.replace(metadata.base, "")
      : event.request.uri;

    // Route to S3 files
    try {
      const u = decodeURIComponent(baselessUri);
      const postfixes = u.endsWith("/")
        ? ["index.html"]
        : ["", ".html", "/index.html"];
      const v = await Promise.any(postfixes.map(p => cf.kvs().get(kvNamespace + ":" + u + p).then(v => p)));
      event.request.uri = metadata.s3.dir + event.request.uri + v;
      setS3Origin(metadata.s3.domain);
      return;
    } catch (e) {}

    // Route to S3 routes
    if (metadata.s3 && metadata.s3.routes) {
      for (var i=0, l=metadata.s3.routes.length; i<l; i++) {
        const route = metadata.s3.routes[i];
        if (baselessUri.startsWith(route)) {
          event.request.uri = metadata.s3.dir + event.request.uri;
          if (event.request.uri.endsWith("/")) {
            event.request.uri += "index.html";
          }
          else if (!event.request.uri.split("/").pop().includes(".")) {
            event.request.uri += "/index.html";
          }
          setS3Origin(metadata.s3.domain);
          return;
        }
      }
    }

    // Route to S3 custom 404 (no servers)
    if (metadata.custom404) {
      event.request.uri = metadata.s3.dir + (metadata.base ? metadata.base : "") + metadata.custom404;
      setS3Origin(metadata.s3.domain);
      return;
    }

    // Route to image optimizer
    if (metadata.image && baselessUri.startsWith(metadata.image.route)) {
      setNextjsCacheKey();
      setUrlOrigin(metadata.image.host, metadata.image.originAccessControlConfig ? { originAccessControlConfig: metadata.image.originAccessControlConfig } : undefined);
      return;
    }

    // Route to servers
    if (metadata.servers){
      event.request.headers["x-forwarded-host"] = event.request.headers.host;
      // In SvelteKit, form action requests contain "/" in request query string.
      // CloudFront does not allow query string with "/". It needs to be encoded.
      for (var key in event.request.querystring) {
        if (key.includes("/")) {
          event.request.querystring[encodeURIComponent(key)] = event.request.querystring[key];
          delete event.request.querystring[key];
        }
      }
      setNextjsGeoHeaders();
      setNextjsCacheKey();
      setUrlOrigin(findNearestServer(metadata.servers), metadata.origin);
    }

    function setNextjsGeoHeaders() {
      if(event.request.headers["cloudfront-viewer-city"]) {
        event.request.headers["x-open-next-city"] = event.request.headers["cloudfront-viewer-city"];
      }
      if(event.request.headers["cloudfront-viewer-country"]) {
        event.request.headers["x-open-next-country"] = event.request.headers["cloudfront-viewer-country"];
      }
      if(event.request.headers["cloudfront-viewer-region"]) {
        event.request.headers["x-open-next-region"] = event.request.headers["cloudfront-viewer-region"];
      }
      if(event.request.headers["cloudfront-viewer-latitude"]) {
        event.request.headers["x-open-next-latitude"] = event.request.headers["cloudfront-viewer-latitude"];
      }
      if(event.request.headers["cloudfront-viewer-longitude"]) {
        event.request.headers["x-open-next-longitude"] = event.request.headers["cloudfront-viewer-longitude"];
      }
    }

    function setNextjsCacheKey() {
      var cacheKey = "";
      if (event.request.uri.startsWith("/_next/image")) {
        cacheKey = getHeader("accept");
      } else {
        cacheKey =
          getHeader("rsc") +
          getHeader("next-router-prefetch") +
          getHeader("next-router-state-tree") +
          getHeader("next-url") +
          getHeader("x-prerender-revalidate");
      }
      if (event.request.cookies["__prerender_bypass"]) {
        cacheKey += event.request.cookies["__prerender_bypass"]
          ? event.request.cookies["__prerender_bypass"].value
          : "";
      }
      var crypto = require("crypto");
      var hashedKey = crypto.createHash("md5").update(cacheKey).digest("hex");
      event.request.headers["x-open-next-cache-key"] = { value: hashedKey };
    }

    function getHeader(key) {
      var header = event.request.headers[key];
      if (header) {
        if (header.multiValue) {
          return header.multiValue.map((header) => header.value).join(",");
        }
        if (header.value) {
          return header.value;
        }
      }
      return "";
    }

    function findNearestServer(servers) {
      if (servers.length === 1) return servers[0][0];

      const h = event.request.headers;
      const lat = h["cloudfront-viewer-latitude"] && h["cloudfront-viewer-latitude"].value;
      const lon = h["cloudfront-viewer-longitude"] && h["cloudfront-viewer-longitude"].value;
      if (!lat || !lon) return servers[0][0];

      return servers
        .map((s) => ({
          distance: haversineDistance(lat, lon, s[1], s[2]),
          host: s[0],
        }))
        .sort((a, b) => a.distance - b.distance)[0]
        .host;
    }

    function haversineDistance(lat1, lon1, lat2, lon2) {
      const toRad = angle => angle * Math.PI / 180;
      const radLat1 = toRad(lat1);
      const radLat2 = toRad(lat2);
      const dLat = toRad(lat2 - lat1);
      const dLon = toRad(lon2 - lon1);
      const a = Math.sin(dLat / 2) ** 2 + Math.cos(radLat1) * Math.cos(radLat2) * Math.sin(dLon / 2) ** 2;
      return 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    }
  }

  function setUrlOrigin(urlHost, override) {
    event.request.headers["x-forwarded-host"] = event.request.headers.host;
    const origin = {
      domainName: urlHost,
      customOriginConfig: {
        port: 443,
        protocol: "https",
        sslProtocols: ["TLSv1.2"],
      },
      originAccessControlConfig: {
        enabled: false,
      }
    };
    override = override ?? {};
    if (override.protocol === "http") {
      delete origin.customOriginConfig;
    }
    if (override.connectionAttempts) {
      origin.connectionAttempts = override.connectionAttempts;
    }
    if (override.timeouts) {
      origin.timeouts = override.timeouts;
    }
    if (override.originAccessControlConfig) {
      origin.originAccessControlConfig = override.originAccessControlConfig;
    }
    cf.updateRequestOrigin(origin);
  }

  function setS3Origin(s3Domain, override) {
    delete event.request.headers["Cookies"];
    delete event.request.headers["cookies"];
    delete event.request.cookies;

    const origin = {
      domainName: s3Domain,
      originAccessControlConfig: {
        enabled: true,
        signingBehavior: "always",
        signingProtocol: "sigv4",
        originType: "s3",
      }
    };
    override = override ?? {};
    if (override.connectionAttempts) {
      origin.connectionAttempts = override.connectionAttempts;
    }
    if (override.timeouts) {
      origin.timeouts = override.timeouts;
    }
    cf.updateRequestOrigin(origin);
  }

  const kvNamespace = "__SST_KV_NAMESPACE__";

  let metadata;
  try {
    const v = await cf.kvs().get(kvNamespace + ":metadata");
    metadata = JSON.parse(v);
  } catch (e) {}

  await routeSite(kvNamespace, metadata);
  return event.request;
}
