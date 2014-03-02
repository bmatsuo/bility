#!/usr/bin/env ruby

require 'csv'
require 'set'

if ARGV.length < 1
  STDERR.puts "usage: #{__FILE__} REPORT"
  exit 1
end

filename = ARGV[0]

types = Set.new

CSV.foreach(filename, :headers => :first_row) do |row|
  next if row["ProductName"] != "Amazon Elastic Compute Cloud"
  next if row["Operation"] != "RunInstances"
  utype = row["UsageType"]
  m = utype.match(/^BoxUsage(?:[:](.*))$/)
  type = m ? m[1] : 'm1.small' # TODO check the description to validate m1.small
  types.add(type)
end

puts types.length
puts types.to_a
