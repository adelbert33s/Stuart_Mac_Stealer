"use strict";

const geoip = require("geoip-lite");

function lookupIp(ip) {
  if (!ip) {
    return { country: null, country_code: null, city: null };
  }

  let clean = ip;
  if (clean.startsWith("::ffff:")) clean = clean.slice(7);

  const geo = geoip.lookup(clean);
  if (!geo) {
    return { country: null, country_code: null, city: null };
  }

  return {
    country: geo.country || null,
    country_code: geo.country || null,
    city: (geo.city || null),
  };
}

function clientIp(req) {
  const forwarded = req.headers["x-forwarded-for"];
  if (forwarded) {
    return String(forwarded).split(",")[0].trim();
  }
  return req.socket?.remoteAddress || req.ip || null;
}

module.exports = { lookupIp, clientIp };