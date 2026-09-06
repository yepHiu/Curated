import { describe,it,expect } from 'vitest'
import { findVisualFramePairs, frameDifferenceHash } from './visual-similarity'
describe('visual similarity review',()=>{
 it('ignores blank images and compares only within the same movie',()=>{
  expect(frameDifferenceHash(new Uint8ClampedArray(288))).toBeNull()
  expect(findVisualFramePairs([{id:'a',movieId:'m',bits:1n},{id:'b',movieId:'m',bits:3n},{id:'c',movieId:'other',bits:1n},{id:'d',movieId:'m',bits:0xffffn}],2)).toEqual([['a','b']])
 })
 it('encodes horizontal luminance differences',()=>{
  const pixels=new Uint8ClampedArray(288)
  for(let y=0;y<8;y++)for(let x=0;x<9;x++){const i=(y*9+x)*4;pixels[i]=pixels[i+1]=pixels[i+2]=255-x*20;pixels[i+3]=255}
  expect(frameDifferenceHash(pixels)).toBe(0xffffffffffffffffn)
 })
})
