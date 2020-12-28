#!/usr/bin/env ruby

File.open('INSTALL.md', 'r') do |f|
  f.each_line do |line|
    forbidden_words = ['Table of contents', 'define', 'pragma']
    next if !line.start_with?('#') || forbidden_words.any? { |w| line =~ /#{w}/ }

    title = line.delete('#').strip
    href = title.tr(' ', '-').downcase
    puts '  ' * (line.count('#') - 1) + "* [#{title}](\##{href})"
  end
end
