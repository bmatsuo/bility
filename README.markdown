Exercises are implemented as stand-alone, unix-friendly command line programs
for ease of use/development. It is probably not ideal as something deployable.
But the functionality is modular enough to rip out and stick behind some
job-queue.

There are no checks for malicious documents that could run the system out of
memory. Depending on the external dependent systems limiting a process memory
may be a decent, albeit heavy-handed, way to mitigate this. 

##Counting EC2 instance types.

###Usage

Go (sloppy instructions)

    go get ./instance_types
    go run instance_types/instance_types.go CSV_FILE

Ruby

    ./instance_types/instance_types.rb CSV_FILE

###Implementation notes

The set of unique instance types is stored in memory. In practice this should
be ok.

###Complexity

There is a constant amount of work done for each row of the CSV document.
So the runtime is linear in the size of the document.

Streaming a CSV document takes space proportial to the size of one row. The set
of instance types is stored as a 'set' (a map in Go). This _should_ only require
space linear in the number of elements (number of distinct instance types). For
malicious documents the numbef of detected instance types is bounded only by the
number of rows. Similarly the size of a row is bounded by document size. In
practice, with wellformed documents, the memory is bounded by a constant.

##Daily cost by tag

###Usage

Go (sloppy instructions)

    go get ./daily_cost
    go run daily_cost/daily_cost.go CSV_File

Ruby

    ./daily_cost/daily_cost.rb CSV_FILE

###Implementation notes

Accumulators for cost by day by tag-value are stored in memory. This should not be
problematic except for extremely large organizations.

In the current implementation, the tag and tag value are part of the key that
indexes cost accumulators. Hopefully AWS has the size on these values bound from
above to some constant, but extremely long tags/values (on the order of row size)
could prove problematic for this implementation. If this were to become a problem,
tag names in keys could be replaced by their column index and values could be hashed
to give all accumulator keys a uniform size.

###Complexity

For each row of the CSV document, the algorithm updates an accumulator in constant
time for each tag (between zero and thirty one times depending billing item's
duration). The number of tags is on the order of the number of columns in the
document. The runtime is proportial to the product of row and column dimensions.

The space necessary to hold all accumulators is the proportional to the number of
tag-values (for each day of the month, the cost for each tag-value). The number of
tag-values is the sum of cardinalities for each tag's value-space. Within a file,
this is on the order of `(# of rows) * (# of tags)`, which is on the order of
`(# of rows) * (# of columns)`. In practice, in a non-malicious file, the number of
tag-values will not come close to the number of cells in a file. Most tags on all
instances would need to be changing to new values near-hourly.

The space requirements for storing accumulator keys (see the implementation notes),
can be reduced to be proportional to `(# of rows) * (# of columns)` and will that
cost comes out in the wash, asymptotically. As implemented the size required is on
the order of the document size in the worst case.

##Instance tag changes

###Usage

    go get ./instance_tag_changes
    go run instance_tag_changes/instance_tag_changes.go CSV_File

###Implementation notes

The underlying data structure is a linked list for each known instance. The list
elements are the tag-value pairs that changed and the timestamp they were changed.

###Complexity

For each tag in each each row, a linked list is traversed to determine if the
row represents a tag change (as far as we have scanned) and insert it if it is.
This is a linear scan that, in the worst case, is the length of the number of
rows scanned (one instance, all rows change all tags). That would make the
runtime `(# of rows)^2 * (# of tags)`. This should not be the case in practice
with well-formed documents. It is unlikely that the majority of tags will change
frequently. And scans are performed from most frequent backward in time. This
appears to reduce the expected length of a traversal on the given test data.

In the best case there is only one instance and all tags are static. This makes the
work done for each row proportional to the number of tags. Thus, the runtime
decreases to `(# of rows) * (# of tags)` or `(# of rows) * (# of columns)`.

The worst/best cases for time is also the worst/best cases for space. In the worst
case, the linked lists will has as many elements as there are rows. So the total
space will be proportional to `(# of rows) * (# of tags)`. In the best case (one
instance, static tags) the space required is proportionl only to the `(# of tags)`.
