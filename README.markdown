Exercises are implemented as stand-alone, unix-friendly command line programs
for ease of use/development. It is probably not ideal as something deployable.
But the functionality could easily be ripped out and stuck behind a job-queue.

There are no checks for malicious documents that could run the system out of
memory. Depending on the external dependent systems limiting a process memory
may be a decent, albeit heavy-handed, way to mitigate this. 

**Note** The Go build instructions are a little sloppy because they assume you
already cloned the repo. You can just `go get` the programs and run them.

    go get github.com/bmatsuo/bility/instance_types
    go get github.com/bmatsuo/bility/daily_costs
    go get github.com/bmatsuo/bility/instance_tag_changes

##Counting EC2 instance types.

###Usage

Go

    go get ./instance_types
    go run instance_types/instance_types.go CSV_FILE

Ruby

    ./instance_types/instance_types.rb CSV_FILE

###Implementation notes

The set of unique instance types is stored in memory. In practice this should
be ok.

###Complexity

There is a constant amount of work done for each row of the CSV document.
So the runtime is always linear in the size of the document.

Streaming a CSV document takes space proportial to the size of one row. The set
of instance types is stored as a 'set' (a map in Go). This _should_ only require
space linear in the number of elements (number of distinct instance types). For
malicious documents the number of detected instance types is bounded only by the
number of rows. In practice, with wellformed documents, the set size is bounded by
a constant and the space requirements are just that of a single row.

All the programs here use a CSV streaming approach and operate on one row at a time.
So, they all have a runtime factor proportional at least the size of the document
and a space requirement of a row. These factors will be ingored in further sections.


##Daily cost by tag

###Usage

Go

    go get ./daily_cost
    go run daily_cost/daily_cost.go CSV_File

Ruby

    ./daily_cost/daily_cost.rb CSV_FILE

###Implementation notes

Accumulators for cost by day by tag-value are stored in memory. This should not be
problematic except for _extremely_ large organizations.

In the current implementation, the tag and tag value are part of the key that
indexes cost accumulators. Hopefully AWS has the size on these values bound from
above to some constant, but extremely long tags/values (on the order of row size)
could prove problematic for this implementation. If this were to become a problem,
tag names in keys could be replaced by their column index and values could be hashed
to give all accumulator keys a uniform size.

###Complexity

For each row of the CSV document, the algorithm updates an accumulator in constant
time for each tag (between zero and thirty one times depending billing item's
duration). The number of tags is bounded by the number of columns in the document.
So in the worst case, runtime is proportial to the product of row and column
dimensions, on the same order of document size. In the best case there are no tags
and there is a constant amount of work on each row.

The space necessary to hold all accumulators is the proportional to the number of
tag-values (for each day of the month, the cost for each tag-value). The number of
tag-values is the sum of cardinalities for each tag's value-space. Within a file,
this is bounded above by `(# of rows) * (# of tags)`, which is on the order of
`(# of rows) * (# of columns)`. In practice, in a non-malicious file, the number of
tag-values will not come close to the number of cells in a file. Most tags on all
instances would need to be changing to new values near-hourly. If there are no tags,
the best case for space as well, the space is bounded by a constant.

##Instance tag changes

###Usage

    go get ./instance_tag_changes
    go run instance_tag_changes/instance_tag_changes.go CSV_File

###Implementation notes

The underlying data structure is a linked list for each known instance. The list
elements are the tag-value pairs that changed and the timestamp they were changed.

###Complexity

For each tag in each each row, a linked list is traversed to determine if the
row represents a tag change (as far as we have scanned) and append/replace/insert
it if it is. This is a linear scan that, in the worst case, is the length of the
number of rows scanned (one instance, all rows change all tags). That would make
the runtime `(# of rows)^2 * (# of tags)`. This should not be the case in practice
with well-formed documents. It is unlikely that the majority of tags will change
frequently. And scans are performed from most frequent moving backward in time.
This appears to reduce the expected length of a traversal on the given test data.

In the best case there is only one instance and all tags are static. This makes the
work done for each row proportional to the number of tags. Thus, the runtime
decreases to `(# of rows) * (# of tags)`.

The worst/best cases for time is also the worst/best cases for space. In the worst
case, the linked lists will have as many elements as there are rows. So the total
space will be proportional to `(# of rows) * (# of tags)`. In the best case (one
instance, static tags) the space required is proportionl only to `(# of tags)`.

##Testing

    go test ./...
