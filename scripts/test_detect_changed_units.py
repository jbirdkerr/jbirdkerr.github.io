import unittest
from scripts.detect_changed_units import (
    matches_patterns,
    detect_affected_units,
    get_topological_order,
)


class TestDetectChangedUnits(unittest.TestCase):
    def setUp(self):
        self.units = {
            "blog": {
                "paths": ["blog/**"],
                "depends_on": [],
            },
            "boopmaster": {
                "paths": ["apps/boopmaster/**"],
                "depends_on": [],
            },
            "db": {
                "paths": ["db/supabase/migrations/**"],
                "depends_on": [],
            },
            "metrics-worker": {
                "paths": ["apps/metrics-worker/**"],
                "depends_on": ["db"],
            },
            "url-shortener": {
                "paths": ["apps/url-shortener/**"],
                "depends_on": ["db"],
            },
        }

    def test_matches_patterns(self):
        self.assertTrue(matches_patterns("blog/src/pages/index.astro", ["blog/**"]))
        self.assertFalse(matches_patterns("apps/boopmaster/index.html", ["blog/**"]))
        self.assertTrue(matches_patterns("db/supabase/migrations/01.sql", ["db/supabase/migrations/**"]))

    def test_detect_affected_units_direct(self):
        changed = ["blog/src/index.astro"]
        directly, affected = detect_affected_units(self.units, changed)
        self.assertEqual(directly, {"blog"})
        self.assertEqual(affected, {"blog"})

    def test_detect_affected_units_transitive(self):
        # When db changes, metrics-worker and url-shortener depend on it
        changed = ["db/supabase/migrations/03_new.sql"]
        directly, affected = detect_affected_units(self.units, changed)
        self.assertEqual(directly, {"db"})
        self.assertEqual(affected, {"db", "metrics-worker", "url-shortener"})

    def test_topological_order(self):
        affected = {"metrics-worker", "db", "url-shortener", "blog"}
        order = get_topological_order(self.units, affected)
        # db must be before metrics-worker and url-shortener
        self.assertLess(order.index("db"), order.index("metrics-worker"))
        self.assertLess(order.index("db"), order.index("url-shortener"))


if __name__ == "__main__":
    unittest.main()
