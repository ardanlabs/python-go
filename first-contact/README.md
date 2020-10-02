# First Contact With Data

> Every single company I've worked at and talked to has the same problem without a single exception so far - poor data quality, especially tracking data. Either there's incomplete data, missing tracking data, duplicative tracking data.
> - DJ Patil

I spend a lot of my time digging in various companies' data. Every time, I am surprised by what I'm seeing there, and every time, the engineers and analysts at the company are surprised as well. I've seen missing data, bad data, data nobody knows what it means and many other oddities.

As a data scientist, the quality of the data you work with is crucial to your success. The old GIGO acronym, which stands for "garbage in, garbage out" is very true. In this blog post we'll discuss some methods and practices that will help you with your first contact with data that will save you a lot of grief down the road.

I'm going to assume you'll be using [pandas](https://pandas.pydata.org/) to process the data. I'll be using pandas version 1.1 and Python 3.5. The code shown here is an [IPython](https://ipython.org/) session.

### Schema

All data have a schema, it is either formally written somewhere or in about 1,000 places in the code. Try to find the schema for the data you're working with, ask people about it - and don't trust it until you compare it with the real data.

Let's have a look at sample weather data from [NOAA](https://www.noaa.gov/weather). The data was originally in CSV format, and I've indented it for readability.

**Listing 1: Weather Data**
```
      DATE  SNOW  TMAX  TMIN  PGTM
2000-01-01     0   100    11  1337
2000-01-02     0   156    61  2313
2000-01-03     0   178   106   320
2000-01-04     0   156    78  1819
2000-01-05     0    83   -17   843
```

Without a schema, it's hard to know what's going on here. The `DATE` column is obvious, but the rest are unclear:

- What is `SNOW`? How much fell? If so is it inches? centimeters? ... Maybe it's a boolean for yes/no?
- `TMAX` and `TMIN` are probably the maximal and minimal temperature at the day. What are the units? Both Celsius and Fahrenheit don't make sense - an 89 difference in one day?
- And what does `PGTM` stand for? What are these numbers?

It's clear that types (string, integer, float ...) are not enough. We need units and maybe more to understand what's going one

**Listing 2: Weather Schema**

```
TMAX - Maximum temperature (tenths of degrees C)
TMIN - Minimum temperature (tenths of degrees C)
SNOW - Snowfall (mm)
PGTM - Peak gust time (hours and minutes, i.e., HHMM)
```

One we read the schema, things become clear. `TMAX` and `TMIN` values make sense, `SNOW` is in millimeters and `PGTM` values are actually time of day.

If you have a say in your company, try to see that all data have a formal written schema, and that this schema is kept up to date.

Even if your company keeps schemas and updates them, data will still have errors. As agent Mulder used to say: "Trust no one!". Always look at the raw data and see that it matches your assumptions about it.

### Size Matters

pandas is built to work in memory and by default will load the whole data into memory. Some datasets are too big to fit in memory, and once you exhaust the computer's physical memory and start to swap to disk - performance goes down the drain.

Load a small part of the data initially, and then extrapolate to guess the final size it'll take in memory. If you're working with a database that's pretty easy - add a `LIMIT` clause to your `SELECT` statement and you're done. If the data is in file - you'll need to work harder.

I'm to look at part of the [NYC Taxi Dataset](https://www1.nyc.gov/site/tlc/about/tlc-trip-record-data.page). The data comes as a compressed CSV and I'd like to find out how much memory it'll take once loaded to a pandas DataFrame.

The first thing I start with is to look at size on disk.

**Listing 3: Disk Size**

```
In [1]: csv_file = 'yellow_tripdata_2018-05.csv.bz2'
In [2]: from pathlib import Path
In [3]: MB = 2**20
In [4]: Path(csv_file).stat().st_size / MB
Out[4]: 85.04909038543701
```

About 85MB compressed on disk. From my experience bz2 compresses text to about 10-15% from its original size. Uncompressed this data will be around 780MB on disk.

The next thing is to find out how many lines there are.

**List 4: Line Count**

```
In [5]: import bz2
In [6]: with bz2.open(csv_file, 'rt') as fp:
   ...:     num_lines = sum(1 for _ in fp)
In [7]: num_lines
Out[7]: 9224065
In [8]: f'{num_lines:,}'
Out[8]: '9,224,065'
```

On `5` we import the `bz2` and on `6` we sum a generator expression to get the number of lines in the file, this took about 40 seconds on my machine.
On `8` I use an `f-string` to print the number in a more human readable format.

We have 9.2 million lines in the file. Let's load 10,000, into a DataFrame, measure the size and calculate the whole size.

**Listing 5: Calculating Size**

```
In [9]: import pandas as pd
In [10]: nrows = 10_000
In [11]: df = pd.read_csv(csv_file, nrows=10_000)
In [12]: df.memory_usage(deep=True).sum() / MB
Out[12]: 3.070953369140625
In [13]: Out[12] * (num_lines / nrows)
Out[13]: 2832.667348892212
```

On `11` we load 10,000 rows to a DataFrame. On `12` we calculate how much memory the DataFrame is consuming in MB and on `13` we calculate the total memory consumption for the whole data - about 2.8GB. The cat can fit in memory.

_Note: If the data doesn't fit in your computer's memory, don't despair! There are way to load parts of data and reduce memory consumption. But probably the most effective solution is to lease a cloud machine with a lot of memory. Some cloud providers have machines with several **terabytes** of memory._

### Raw Data

Before you load the data, it's a good idea to look at it in it's raw format and see if it matches your assumptions about it.

**Listing 6: Raw Data**

```
In [14]: with bz2.open(csv_file, 'rt') as fp:
    ...:     for i, line in enumerate(fp):
    ...:         print(line.strip())
    ...:         if i == 3:
    ...:             break
    ...: 
VendorID,tpep_pickup_datetime,tpep_dropoff_datetime,passenger_count,trip_distance,RatecodeID,store_and_fwd_flag,PULocationID,DOLocationID,payment_type,fare_amount,extra,mta_tax,tip_amount,tolls_amount,improvement_surcharge,total_amount

1,2018-05-01 00:13:56,2018-05-01 00:22:46,1,1.60,1,N,230,50,1,8,0.5,0.5,1.85,0,0.3,11.15
1,2018-05-01 00:23:26,2018-05-01 00:29:56,1,1.70,1,N,263,239,1,7.5,0.5,0.5,2,0,0.3,10.8
```

Looks like a CSV with a header line. Let's use the `csv` module to get rows.

**Listing 7: Raw Data - CSV**

```
In [15]: from pprint import pprint
In [16]: with bz2.open(csv_file, 'rt') as fp:
    ...:     rdr = csv.DictReader(fp)
    ...:     for i, row in enumerate(rdr):
    ...:         pprint(row)
    ...:         if i == 3:
    ...:             break
    ...: 
{'DOLocationID': '50',
 'PULocationID': '230',
 'RatecodeID': '1',
 'VendorID': '1',
 'extra': '0.5',
 'fare_amount': '8',
 'improvement_surcharge': '0.3',
 'mta_tax': '0.5',
 'passenger_count': '1',
 'payment_type': '1',
 'store_and_fwd_flag': 'N',
 'tip_amount': '1.85',
 'tolls_amount': '0',
 'total_amount': '11.15',
 'tpep_dropoff_datetime': '2018-05-01 00:22:46',
 'tpep_pickup_datetime': '2018-05-01 00:13:56',
 'trip_distance': '1.60'}
...
```

On `15` we load the `pprint` module for more human readable printing. Then we use `csv.DictReader` to read 3 records and print them. Looking at the data it seems OK: datetime fields look like data & time, amounts look like floating point numbers etc.

### Data Types

Once you see the raw data, and verify you can load the data to memory - you can load the data into a DataFrame. However remember that in CSV everything is text and pandas is guessing the types for you - so you need to check it.


**Listing 8: Checking Types**

```
In [16]: df = pd.read_csv(csv_file)
In [17]: df.dtypes
Out[17]: 
VendorID                   int64
tpep_pickup_datetime      object
tpep_dropoff_datetime     object
passenger_count            int64
trip_distance            float64
RatecodeID                 int64
store_and_fwd_flag        object
PULocationID               int64
DOLocationID               int64
payment_type               int64
fare_amount              float64
extra                    float64
mta_tax                  float64
tip_amount               float64
tolls_amount             float64
improvement_surcharge    float64
total_amount             float64
dtype: object
```

On `16` we load the whole data into a DataFrame, this took about 45 seconds on my machine. On `17` we print out the data type for each column.

Most of the column types seem OK, but `tpep_pickup_datetime` and `tpep_dropoff_datetime` are `object`. The `object` type usually means a string, and we'd like to have a time stamp here. This is a case where pandas need some help figuring out types.

_Note: I hate the CSV format with a passion - there's no type information, no formal specification, and don't get me started on Unicode ... If you have a say - pick a different format which has type information. My default storage format is [SQlite](https://www.sqlite.org/) which is a one-file SQL database._

Let's help pandas figure out the types.

**Listing 9: Fixing Types**

```
In [18]: time_cols = ['tpep_pickup_datetime', 'tpep_dropoff_datetime']
In [19]: df = pd.read_csv(csv_file, parse_dates=time_cols)
In [20]: df.dtypes
Out[20]: 
VendorID                          int64
tpep_pickup_datetime     datetime64[ns]
tpep_dropoff_datetime    datetime64[ns]
passenger_count                   int64
trip_distance                   float64
RatecodeID                        int64
store_and_fwd_flag               object
PULocationID                      int64
DOLocationID                      int64
payment_type                      int64
fare_amount                     float64
extra                           float64
mta_tax                         float64
tip_amount                      float64
tolls_amount                    float64
improvement_surcharge           float64
total_amount                    float64
dtype: object
```

On `19` we tell pandas to parse the two time columns as dates. And by looking at the output of `20` we see now that we have the right types.

### Looking for Outliers

Once the data is loaded and in the right types, it's time to look for bad values. The definition of "bad data" depends on the data you're working with. For example if you have a `temperature` column, the maximal value probably shouldn't be more than 60°C (the highest temperature ever recorded was 56.7°C). But - what if we're talking about engine temperature?

One of the easy ways to start is to use the DataFrame's `describe` method. Since our DataFrame has many columns, I'm going to look at a subset of the columns.

**Listing 10: Looking for Outliers**

```
In [21]: df[['trip_distance', 'total_amount', 'passenger_count']].describe()
Out[21]: 
       trip_distance  total_amount  passenger_count
count   9.224063e+06  9.224063e+06     9.224063e+06
mean    3.014031e+00  1.681252e+01     1.596710e+00
std     3.886332e+00  7.864489e+01     1.245703e+00
min     0.000000e+00 -4.858000e+02     0.000000e+00
25%     1.000000e+00  8.760000e+00     1.000000e+00
50%     1.650000e+00  1.225000e+01     1.000000e+00
75%     3.100000e+00  1.835000e+01     2.000000e+00
max     9.108000e+02  2.346327e+05     9.000000e+00
```

Right away we see some fishy data:

- The minimal `total_amount` is negative
- The maximal `trip_distance` is 910 miles
- The are rides with 0 passengers

Sometimes you'll need to run a calculation to find outliers

**Listing 11: Trip Duration**

```
In [22]: (df['tpep_dropoff_datetime'] - df['tpep_pickup_datetime']).describe()
Out[22]: 
count                      9224063
mean     0 days 00:16:29.874877155
std      2 days 10:35:51.816665095
min           -7410 days +13:10:08
25%                0 days 00:06:46
50%                0 days 00:11:28
75%                0 days 00:19:01
max                0 days 23:59:59
dtype: object
```

On `22` we calculate the trip duration and use `describe` to display statistics on it. 

- The minimal duration is negative (maybe someone invented a time machine?)
- The maximal duration is a full day

### Conclusion

I haven't met real data that didn't have errors in it. I've learned to keep my eyes open and challenge everything I *think* I know about the data before starting to process it. I urge you to follow these steps every time you start working with new data:

- Find out the schema
- Calculate data size
- Look at the raw data
- Check data types
- Look for outliers

This might seem like a lot of work, but I guarantee it'll save you much more work down the road when the models you’ve worked hard to develop start to misbehave.

I'd love to hear your data horror stories, and how you handled them. Reach out to me at miki@353solutions and amaze me.


