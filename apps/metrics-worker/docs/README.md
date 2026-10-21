# Documentation Index

Welcome to the **jbirdkerr.net** technical documentation. This site pairs an interactive static frontend with a real-time event analytics and persistence pipeline powered by Cloudflare Workers (Python), Supabase (PostgreSQL), and PostHog.

---

## 📚 Topics

- [**System Architecture**](architecture.md)
  High-level architecture diagram and component breakdown across Client, Cloudflare Edge, Supabase, and PostHog.

- [**Metrics & Sequence Flows**](metrics-pipeline.md)
  Deep-dive sequence diagrams and explanations for the dual-ingestion pipeline and live HTMX metrics dashboard rendering.

- [**Worker API Reference**](api.md)
  Detailed specification of all Cloudflare Worker REST endpoints, caching strategies, payloads, and secrets.

- [**Database Schema & Migrations**](../supabase/migrations/20260923042823_01-initial-schema.sql)
  PostgreSQL schemas, tables, views, and atomic RPC functions for metric aggregation.
