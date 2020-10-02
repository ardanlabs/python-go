# First Contact With Data

> Every single company I've worked at and talked to has the same problem without a single exception so far - poor data quality, especially tracking data. Either there's incomplete data, missing tracking data, duplicative tracking data.
> - DJ Patil

I spend a lot of my time digging in various companies data. Every time, I am surprised of what I'm seeing there, and every time, the engineers and analysts at the company are surprised as well.

As a data scientist, the quality of the data you work with is crucial to your success. The old GIGO acronym, which stands for "garbage in, garbage out" is very true. In this blog post we'll discuss some methods and practices that will help you with your first contact with data that will save you a lot of grief down the road.

I'm going to assume you'll be using [pandas](https://pandas.pydata.org/) to process the data. I'll be using pandas version 1.1 and Python 3.5.

### Size Matters

pandas is build to work in memory and by default will load the whole data into memory. 
