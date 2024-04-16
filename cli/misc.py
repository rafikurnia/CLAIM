#!/usr/bin/env python3

import json
import sys
import csv


if __name__ == "__main__":
    with open(sys.argv[1]) as f:
        results = json.load(f)["results"]

        regions = list(results.keys())
        fields = list(results[regions[0]]["1"]["By"].keys())

        print("region,{}".format(",".join(fields)))
        for r in regions:
            str = r
            for f in fields:
                str += ",{}".format(results[r]["1"]["By"][f])
            print(str)
