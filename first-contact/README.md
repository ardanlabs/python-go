# First Contact With Data

### Introduction

_Every single company I've worked at and talked to has the same problem without a single exception so far - poor data quality, especially tracking data. Either there's incomplete data, missing tracking data, duplicative tracking data._ - DJ Patil

I spend a lot of my time digging into data at various companies. Most of the time I’m surprised by what I see and so are the engineers and analysts that work at these companies. I've seen missing data, bad data, data nobody knows anything about, and many other oddities.

As a data scientist, the quality of the data you work with is crucial to your success. The old GIGO acronym, which stands for "garbage in, garbage out", is very true. In this blog post, we'll discuss methods and practices that will make your first contact with data successful and will save you from a lot of grief down the road.

I'll be using Python 3.8 and [pandas](https://pandas.pydata.org/) version 1.1 in this post. The code shown here is an [IPython](https://ipython.org/) session.

### Schema

*All* data has a schema in one way or the other. Sometimes it’s properly documented and sometimes it’s spread out in a thousand different places all over the code. It’s important to find this documentation and don't trust it until you compare it with the real data.

Let's have a look at weather data from [NOAA](https://www.noaa.gov/weather). The data is provided by NOAA in a CSV format, but I've changed the format here for readability.

**Listing 1: Weather Data**
```
      DATE      SNOW      TMAX      TMIN      PGTM
2000-01-01         0       100        11      1337
2000-01-02         0       156        61      2313
2000-01-03         0       178       106       320
2000-01-04         0       156        78      1819
2000-01-05         0        83       -17       843
```

Without a properly documented schema, it's hard to know what's going on. The `DATE` column is obvious, but it’s unclear what the rest of the fields mean.

- What does `SNOW` represent? Is it describing how much snow fell? If so, is that measurement in inches or maybe centimeters? Maybe it's a boolean value for representing yes or no?
- `TMAX` and `TMIN` are probably the maximum and minimum temperature for the day. Once again, what are the units since Celsius and Fahrenheit don't make sense being there is an 89 degree difference in 2000-01-01?
- I have no clue what `PGTM` stands for? 

It's clear that only knowing the type of the data (string, integer, float ...) isn’t enough. We need to know the units and maybe more to understand the information..

**Listing 2: Weather Schema**

```
TMAX - Maximum temperature (tenths of degrees C)
TMIN - Minimum temperature (tenths of degrees C)
SNOW - Snowfall (mm)
PGTM - Peak gust time (hours and minutes, i.e., HHMM)
```

NOAA did publish a schema and after reading it, the representation of the data is now clear. `TMAX` and `TMIN` are Celsius temperatures but in tenths of a degree. `SNOW` represents snowfall, but in millimeters. Finally,  `PGTM` represents time values for when the peak wind gust occured.

If you have a say in your company, do your best to make sure all the data you are working with has a formal documented written schema, and that this schema is kept up to date.

Even if your company maintains schemas and keeps them up to date, the raw data you’re processing can still have errors. Always look at the raw data and check if it matches your documented schemas. As agent Mulder said: "Trust no one!".

### Size Matters

pandas by default will load all the data you need to work with into memory. Some datasets are too big to fit all of it in memory, and once you exhaust the computer's physical memory and start to swap to disk, performance processing the data goes down the drain.

My advice is to load a small amount of the data into memory initially, and then extrapolate to figure out how much memory you will need for the entire dataset. If the dataset is coming from a database, add a `LIMIT` clause to your `SELECT` statement to reduce the initial size of the dataset. If the dataset is coming from a file, you'll need a different strategy like reading a limited number of lines of text or a limited number of parsable documents.

Let’s have a look at a different dataset that is part of the [NYC Taxi Dataset](https://www1.nyc.gov/site/tlc/about/tlc-trip-record-data.page). This data comes as a compressed CSV and I'd like to know how much memory I’ll need to load all of it in pandas.

First, let’s look at how large the dataset is on disk.

**Listing 3: Disk Size**

```
In [1]: csv_file = 'yellow_tripdata_2018-05.csv.bz2'
In [2]: from pathlib import Path
In [3]: MB = 2**20
In [4]: Path(csv_file).stat().st_size / MB
Out[4]: 85.04909038543701
```

The compressed CSV is consuming about 85MB of disk space. From my experience, `bz2` compressed text is about 10-15% smaller from its original size, which means the uncompressed data will consume around 780MB of disk space.

The next thing to know is how many lines of text are there in the file.

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

* `[5]` we import the `bz2` library
* `[6]` we use a generator expression to count the number of lines of text in the file. This took about 40 seconds to run on my machine.
* `[8]` I use an `f-string` to print the result in a human readable format.

We have 9.2 million lines of text in the file. Let's load the first 10,000 lines into pandas and measure the amount of memory being used. Then we can calculate the total amount of memory needed to load the complete file.

**Listing 5: Calculating Size**

```
In [9]: import pandas as pd
In [10]: nrows = 10_000
In [11]: df = pd.read_csv(csv_file, nrows=nrows)
In [12]: df.memory_usage(deep=True).sum() / MB
Out[12]: 3.070953369140625
In [13]: Out[12] * (num_lines / nrows)
Out[13]: 2832.667348892212
```

* `[11]` we load 10,000 rows to a DataFrame.
* `[12]` we calculate how much memory the DataFrame is consuming in MB.
* `[13]` we calculate the total memory consumption for the whole data file. ~2.8GB

It’s safe to load all of the data into memory.

_Note: If the data doesn't fit in your computer's memory, don't despair! There are ways to load parts of data and reduce memory consumption. But probably the most effective solution is to lease a cloud machine with a lot of memory. Some cloud providers have machines with several **terabytes** of memory._

### Raw Data

Before you load any data, it's a good idea to look at it in it's raw format and see if it matches your understanding of the schema.

**Listing 6: Raw Data**

```
In [14]: with bz2.open(csv_file, 'rt') as fp:
    ...:     for i, line in enumerate(fp):
    ...:         print(line.strip())
    ...:         if i == 3:
    ...:             break
    ...:

Output:
VendorID,tpep_pickup_datetime,tpep_dropoff_datetime,passenger_count,trip_distance,RatecodeID,store_and_fwd_flag,PULocationID,DOLocationID,payment_type,fare_amount,extra,mta_tax,tip_amount,tolls_amount,improvement_surcharge,total_amount

1,2018-05-01 00:13:56,2018-05-01 00:22:46,1,1.60,1,N,230,50,1,8,0.5,0.5,1.85,0,0.3,11.15
1,2018-05-01 00:23:26,2018-05-01 00:29:56,1,1.70,1,N,263,239,1,7.5,0.5,0.5,2,0,0.3,10.8
```

It looks like this file is a CSV with a header on the first line. Let's use the `csv` module to get a rows count.

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

Output:
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

* `[15]` load the `pprint` module for human readable printing.
* `[16]` use a `csv.DictReader` to read 3 records and print them.

Looking at the data, things seem OK. The datetime fields look like date and time, also the amounts look like floating point numbers.

### Data Types

Once you see the raw data and verify you can load the data into memory, you can load the data into pandas. However, remember that in CSV everything is text and pandas is guessing the types for you. After loading the data, check that column types match what you expect.

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

* `[16]` we load the whole data into a DataFrame, this took about 45 seconds on my machine.
* `[17]` we print out the data type for each column.

Most of the column types seem OK, but `tpep_pickup_datetime` and `tpep_dropoff_datetime` are of type `object`. The `object` type usually means a string, but in this case we'd like these columns to be a timestamp. This is a case where we need to help pandas figure out the correct type for these columns.

_Note: I hate the CSV format with a passion - there's no type information, no formal specification, and don't get me started on Unicode ... If you have a say - pick a different format which has type information. My default storage format is [SQlite](https://www.sqlite.org/) which is a one-file SQL database._

Let's help pandas figure out the correct types.

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

* `[19]` we tell pandas to parse the two time columns as dates. Looking at the output of `[20]`, we see we now have the right types.

### Looking for Bad Values

Once the data is loaded and the correct types are being used, it's time to look for “bad” values. The definition of a "bad” value depends on the data you're working with. For example, if you have a `temperature` column, the maximal value probably shouldn't be more than 60°C (the highest temperature ever recorded on earth was 56.7°C). However, if the data represents engine temperatures, the “bad” value markers would need to be much higher.

One of the easiest ways to look for bad data is to use the DataFrame's `describe` method. Since our DataFrame has many columns, I'm going to look at a subset of the columns.

**Listing 10: Looking for Bad Data**

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

* The minimal `total_amount` is negative
* The maximal `trip_distance` is 910 miles
* There are rides with 0 passengers

Sometimes you'll need to run a calculation to find bad data.

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

* `[22]` we calculate the trip duration and use `describe` to display statistics on it. 

* The minimal duration is negative (maybe someone invented a time machine?)
* The maximal duration is a full day

In some cases, you’ll need to run more complex queries to find bad data. For example - the speed can’t be more than 55mph. Or, if you look at the weather data, there shouldn’t be any snow when the temperature is above 20°C.

### Conclusion

I haven't worked with real data that didn't have errors in it. I've learned to keep my eyes open and challenge everything I *think* I know about the data before processing it to make decisions. I urge you to follow these steps every time you start working with thinka new dataset:

* Find out the schema
* Calculate data size
* Look at the raw data
* Check data types
* Look for bad data

This might seem like a lot of work, but I guarantee it'll save you much more work down the road when the models you’ve worked hard to develop start to misbehave.

I'd love to hear your data horror stories, and how you handled them. Reach out to me at miki@353solutions and amaze me.

