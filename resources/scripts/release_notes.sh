#!/bin/sh
cat CHANGELOG.md | awk '
  /^## [0-9]/ {
      release++;
  }
  !/^## [0-9]/ {
      if ( release == 1 ) print;
      if ( release > 1 ) exit;
  }'