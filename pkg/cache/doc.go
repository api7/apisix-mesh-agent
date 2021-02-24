// Package cache provides cache solutions to store APISIX resources.
// A cache solution should support to insert, update, get, list and delete
// for each resources. To reduce the type assertion overheads, the cache
// is designed to be typed. Also, the cache should be threaded-safe.
package cache
