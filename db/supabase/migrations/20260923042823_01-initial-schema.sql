SET local check_function_bodies = off;

CREATE TABLE "public"."site_event_counts" (
  "event" text   NOT NULL,
  "count" bigint NOT NULL DEFAULT 0,
  CONSTRAINT "site_event_counts_pkey" PRIMARY KEY (EVENT)
);

ALTER TABLE "public"."site_event_counts"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."site_events" (
  "id"          bigint                   GENERATED ALWAYS AS IDENTITY NOT NULL,
  "occurred_at" timestamp with time zone NOT NULL DEFAULT now(),
  "event"       text                     NOT NULL,
  "distinct_id" text,
  "properties"  jsonb                    NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT "site_events_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."site_events"
  ENABLE ROW LEVEL SECURITY;

CREATE TABLE "public"."site_metrics" (
  "id"                   boolean                  NOT NULL DEFAULT true,
  "total_boops"          bigint                   NOT NULL DEFAULT 0,
  "max_boop_combo"       bigint                   NOT NULL DEFAULT 0,
  "total_events"         bigint                   NOT NULL DEFAULT 0,
  "eye_tracking_toggles" bigint                   NOT NULL DEFAULT 0,
  "googly_eyes_toggles"  bigint                   NOT NULL DEFAULT 0,
  "party_mode_toggles"   bigint                   NOT NULL DEFAULT 0,
  "treat_showers"        bigint                   NOT NULL DEFAULT 0,
  "total_eye_distance"   bigint                   NOT NULL DEFAULT 0,
  "updated_at"           timestamp with time zone NOT NULL DEFAULT now(),
  CONSTRAINT "site_metrics_id_check" CHECK ((id = true)),
  CONSTRAINT "site_metrics_pkey" PRIMARY KEY (id)
);

ALTER TABLE "public"."site_metrics"
  ENABLE ROW LEVEL SECURITY;

ALTER TABLE "public"."site_metrics"
  REPLICA IDENTITY FULL;

CREATE OR REPLACE FUNCTION public.record_site_events (
  p_events jsonb
)
  RETURNS jsonb
  LANGUAGE plpgsql
  SECURITY DEFINER
  SET search_path TO 'public'
  AS $function$
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
$function$;

CREATE INDEX site_events_event_idx ON public.site_events USING btree (EVENT);

CREATE INDEX site_events_occurred_at_idx ON public.site_events USING btree (occurred_at);

CREATE POLICY "public can read site metrics" ON "public"."site_metrics"
  FOR SELECT
  TO "anon", "authenticated"
  USING (true);

ALTER PUBLICATION "supabase_realtime" ADD TABLE "public"."site_metrics";

REVOKE ALL ON FUNCTION "public"."record_site_events"(jsonb) FROM PUBLIC;

GRANT EXECUTE ON FUNCTION "public"."record_site_events"(jsonb) TO "postgres", "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."site_event_counts" TO "anon", "authenticated", "postgres", "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."site_events" TO "anon", "authenticated", "postgres", "service_role";

GRANT DELETE, INSERT, MAINTAIN, REFERENCES, SELECT, TRIGGER, TRUNCATE, UPDATE ON TABLE "public"."site_metrics" TO "anon", "authenticated", "postgres", "service_role";
