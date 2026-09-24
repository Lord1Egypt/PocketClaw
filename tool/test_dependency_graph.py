#!/usr/bin/env python3
"""Tests for the resolved-dependency-graph check."""

import sys
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO / "tool"))

import dependency_graph as graph  # noqa: E402

# The shape `gradle :app:dependencies` prints, including the Umeng lines the
# release compile classpath carried before Phase B.
TREE = r"""
releaseCompileClasspath - Compile classpath for 'release'.
+--- com.umeng.umsdk:common:9.9.1
+--- com.umeng.umsdk:asms:1.8.7.2
+--- androidx.core:core-ktx:1.18.0
|    +--- androidx.annotation:annotation:1.8.1 -> 1.9.1
|    \--- org.jetbrains.kotlin:kotlin-stdlib:2.2.20 (*)
\--- io.flutter:flutter_embedding_release:1.0.0-5d531788691ec3404cac0cee66ead4007b177363
     \--- androidx.lifecycle:lifecycle-common:2.9.4 (c)
"""


class DependencyGraphTest(unittest.TestCase):
    def test_every_coordinate_is_read_at_any_depth(self):
        coords = graph.coordinates(TREE)
        for expected in (("com.umeng.umsdk", "common"), ("androidx.annotation", "annotation"),
                         ("org.jetbrains.kotlin", "kotlin-stdlib"),
                         ("androidx.lifecycle", "lifecycle-common")):
            self.assertIn(expected, coords)

    def test_a_compile_only_proprietary_sdk_is_caught(self):
        self.assertEqual(graph.forbidden(graph.coordinates(TREE)),
                         ["com.umeng.umsdk:asms", "com.umeng.umsdk:common"])

    def test_a_group_matches_itself_and_subgroups_only(self):
        self.assertEqual(graph.forbidden({("com.google.firebase", "firebase-analytics"),
                                          ("com.google.android.gms", "play-services-base")}),
                         ["com.google.android.gms:play-services-base",
                          "com.google.firebase:firebase-analytics"])
        self.assertEqual(graph.forbidden({("com.umengx", "x"), ("com.google.guava", "guava"),
                                          ("androidx.core", "core")}), [])


class BackgroundSchedulerTest(unittest.TestCase):
    def test_workmanager_is_refused_because_it_starts_the_app_at_boot(self):
        hits = graph.forbidden({("androidx.work", "work-runtime"), ("androidx.core", "core")})
        self.assertEqual(hits, ["androidx.work:work-runtime (background scheduler)"])


if __name__ == "__main__":
    unittest.main()
