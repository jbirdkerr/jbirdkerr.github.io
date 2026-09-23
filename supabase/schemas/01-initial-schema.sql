-- jbirdkerr.net realtime metrics
--
-- Run this in the Supabase SQL editor.
--
-- This intentionally keeps the raw PostHog-derived event stream separate
-- from the small, public metrics tables used by the site.

create table if not exists public.site_events (
    id bigint generated always as identity primary key,
    occurred_at timestamptz not null default now(),
    event text not null,
    distinct_id text,
    properties jsonb not null default '{}'::jsonb
);

create index if not exists site_events_occurred_at_idx
    on public.site_events (occurred_at);

create index if not exists site_events_event_idx
    on public.site_events (event);

create table if not exists public.site_event_counts (
    event text primary key,
    count bigint not null default 0
);

create table if not exists public.site_metrics (
    id boolean primary key default true check (id = true),
    total_boops bigint not null default 0,
    max_boop_combo bigint not null default 0,
    total_events bigint not null default 0,
    eye_tracking_toggles bigint not null default 0,
    googly_eyes_toggles bigint not null default 0,
    party_mode_toggles bigint not null default 0,
    treat_showers bigint not null default 0,
    total_eye_distance bigint not null default 0,
    updated_at timestamptz not null default now()
);

-- Browser clients may read the aggregate metrics, but not the raw event table.
alter table public.site_events enable row level security;
alter table public.site_event_counts enable row level security;
alter table public.site_metrics enable row level security;

drop policy if exists "public can read site metrics" on public.site_metrics;
create policy "public can read site metrics"
    on public.site_metrics
    for select
    to anon, authenticated
    using (true);

-- Do not create SELECT policies for site_events or site_event_counts.
-- The Worker uses the service-role key server-side.

create or replace function public.record_site_events(
    p_events jsonb
)
returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
    item jsonb;
    event_name text;
    event_properties jsonb;
    event_time timestamptz;
    event_distinct_id text;
    boop_combo bigint;
    eye_distance bigint;
    feature text;
    enabled boolean;
    current_metrics public.site_metrics;
begin
    if jsonb_typeof(p_events) <> 'array' then
        raise exception 'p_events must be a JSON array';
    end if;

    for item in select value from jsonb_array_elements(p_events)
    loop
        event_name := nullif(item->>'event', '');
        if event_name is null then
            continue;
        end if;

        event_properties := coalesce(item->'properties', '{}'::jsonb);

        begin
            event_time := coalesce(
                (item->>'occurred_at')::timestamptz,
                now()
            );
        exception when others then
            event_time := now();
        end;

        event_distinct_id := nullif(item->>'distinct_id', '');

        insert into public.site_events (
            occurred_at,
            event,
            distinct_id,
            properties
        )
        values (
            event_time,
            event_name,
            event_distinct_id,
            event_properties
        );

        insert into public.site_event_counts (event, count)
        values (event_name, 1)
        on conflict (event)
        do update set count = site_event_counts.count + 1;

        update public.site_metrics
        set
            total_events = total_events + 1,
            total_boops = total_boops +
                case when event_name = 'boop' then 1 else 0 end,
            max_boop_combo =
                case
                    when event_name = 'boop' then greatest(
                        max_boop_combo,
                        coalesce(nullif(event_properties->>'combo_count', '')::bigint, 0)
                    )
                    else max_boop_combo
                end,
            eye_tracking_toggles = eye_tracking_toggles +
                case
                    when event_name = 'feature_toggle'
                     and event_properties->>'feature' = 'eye_tracking'
                     and coalesce((event_properties->>'enabled')::boolean, false)
                    then 1 else 0
                end,
            googly_eyes_toggles = googly_eyes_toggles +
                case
                    when event_name = 'feature_toggle'
                     and event_properties->>'feature' = 'googly_eyes'
                     and coalesce((event_properties->>'enabled')::boolean, false)
                    then 1 else 0
                end,
            party_mode_toggles = party_mode_toggles +
                case
                    when event_name = 'feature_toggle'
                     and event_properties->>'feature' = 'party_mode'
                     and coalesce((event_properties->>'enabled')::boolean, false)
                    then 1 else 0
                end,
            treat_showers = treat_showers +
                case when event_name = 'treat_shower' then 1 else 0 end,
            total_eye_distance = total_eye_distance +
                case
                    when event_name = 'eye_movement' then
                        coalesce(nullif(event_properties->>'distance_px', '')::bigint, 0)
                    else 0
                end,
            updated_at = now()
        where id = true;
    end loop;

    select * into current_metrics
    from public.site_metrics
    where id = true;

    return to_jsonb(current_metrics);
end;
$$;

-- Only the service role should call the ingestion function.
revoke all on function public.record_site_events(jsonb) from public;
revoke all on function public.record_site_events(jsonb) from anon;
revoke all on function public.record_site_events(jsonb) from authenticated;
grant execute on function public.record_site_events(jsonb) to service_role;

-- Enable Supabase Realtime for the small aggregate row.
do $$
begin
    alter publication supabase_realtime add table public.site_metrics;
exception
    when duplicate_object then null;
end
$$;

-- Optional but useful for UPDATE payloads.
alter table public.site_metrics replica identity full;
