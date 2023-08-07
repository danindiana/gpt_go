#!/usr/bin/perl
use strict;
use warnings;

# Prompt the user for the input file
print "Enter the path to the input text file: ";
my $input_file = <STDIN>;
chomp $input_file;

# Check if the input file exists
unless (-e $input_file) {
    die "Input file not found: $input_file\n";
}

# Open the input file for reading
open(my $fh, '<', $input_file) or die "Cannot open file: $input_file\n";

# Read each line of the input file and remove the numbering scheme
while (my $line = <$fh>) {
    chomp $line;
    if ($line =~ m/^\d+\.\s*(.*)$/) {
        print "$1\n";
    }
}

# Close the input file
close($fh);

print "Numbering scheme removed. Finished.\n";
