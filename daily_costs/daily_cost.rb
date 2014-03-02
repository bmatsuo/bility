#!/usr/bin/env ruby

require 'csv'
require 'time'

def main
  if ARGV.length < 1
    STDERR.puts "usage: #{__FILE__} REPORT"
    exit 1
  end

  filename = ARGV[0]

  summarize(filename)
end

def summarize(filename)
  costs = Hash.new(0.0)

  CSV.foreach(filename, :headers => :first_row) do |row|
    breakdown_daily(row).each_pair do |date, cost|
      # aggregate cost by tag
      row.each do |col, value|
        m = col.match(/^user:(.*)$/)
        next if !m
        tag = m[1]
        costs[[date, tag, value]] += cost
      end
    end
  end

  puts "Date,Tag,Value,Cost"
  costs.each_pair do |triple, cost|
    data = triple + [cost]
    puts data.to_csv(:force_quotes => true)
  end
end

# daily costs for a row describing a arbitrary (closed) time span. cost is
# distributed proportionally across each day.
def breakdown_daily(row)
  date_cost = {}
  cost = row["UnBlendedCost"].to_f rescue nil
  return date_cost if !cost # no cost.. no daily cost..
  start_time, end_time = ['UsageStartDate', 'UsageEndDate'].map do |col|
    DateTime.parse(row[col]) rescue nil
  end
  return date_cost if !start_time || !end_time # infinite spans have daily cost of $0.00...
  if start_time > end_time
    raise Exception, "invalid usage window; #{row}"
  end
  window_span = end_time - start_time
  start_date, end_date = [start_time, end_time].map { |t| t.to_date }
  date_cost = {}
  date = start_date
  span_start = start_time
  while date <= end_date do
    span_start = if date == start_date then start_time else date.to_datetime end
    span_stop = if date == end_date then end_time else (date + 1).to_datetime end
    span = span_stop - span_start
    frac = span / window_span
    date_cost[date] = frac * cost

    # iterate..
    date += 1
    span_start = date.to_datetime
  end

  date_cost
end

main
